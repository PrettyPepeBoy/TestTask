package parser

import (
	"context"
	"errors"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"sync"
	"testTask/internal/database"
	"testTask/internal/models"
	"time"
)

var (
	ErrUrlIsEmpty         = errors.New("url is empty")
	ErrTitleIsEmpty       = errors.New("title is empty")
	ErrUsernameIsEmpty    = errors.New("username is empty")
	ErrUsernameUrlIsEmpty = errors.New("usernameUrl is empty")
	ErrDateIsEmpty        = errors.New("date is empty")
)

type Parser struct {
	habs        []*hab
	articlesBuf []*models.ArticleData
	storage     *database.Database

	mx               sync.Mutex
	stop             context.CancelFunc
	ctx              context.Context
	goroutinesAmount int
	c                <-chan articleInfo
}

func NewParser(db *database.Database) (*Parser, error) {
	habsInfo, err := db.GetHabsInfo()
	if err != nil {
		return nil, err
	}

	c := make(chan articleInfo)

	habs := make([]*hab, 0, len(habsInfo))
	for _, habInfo := range habsInfo {
		habs = append(habs, newHab(habInfo, c))
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	p := &Parser{
		articlesBuf:      make([]*models.ArticleData, 0),
		ctx:              ctx,
		stop:             cancel,
		habs:             habs,
		storage:          db,
		goroutinesAmount: viper.GetInt("parser.goroutines-amount"),
		c:                c,
	}

	go func() {
		for {
			time.Sleep(30 * time.Second)
			logrus.Info("start put data in table")
			_ = p.putArticleInTable()
		}
	}()

	return p, nil
}

func (p *Parser) Parse() {
	for _, h := range p.habs {
		go func(h *hab) {
			for {
				select {
				case <-h.timer.C:
					h.parseMainPage()
					//h.timer.Reset(h.interval)
					h.timer.Reset(time.Hour)
				case <-p.ctx.Done():
					return
				}
			}
		}(h)

	}

	for i := 0; i < p.goroutinesAmount; i++ {
		go p.processRoutine(p.ctx)
	}
}

func (p *Parser) AddHub() {

}

func (p *Parser) UnsafeStopParse() {
	p.stop()
}

func (p *Parser) parseArticle(info articleInfo) *models.ArticleData {
	collector := colly.NewCollector()

	var data models.ArticleData

	data.Url = info.url

	collector.OnHTML(info.h.habInfo.ArticlePageQueryUserLink, func(htmlElement *colly.HTMLElement) {
		data.UsernameUrl = htmlElement.Attr("href")
		data.Username = htmlElement.Text
	})

	collector.OnHTML(info.h.habInfo.ArticlePageQueryTitle, func(htmlElement *colly.HTMLElement) {
		data.Title = htmlElement.Text
	})

	collector.OnHTML(info.h.habInfo.ArticlePageQueryTime, func(htmlElement *colly.HTMLElement) {
		data.PublishData, _ = htmlElement.DOM.Children().Attr("datetime")
	})

	err := collector.Visit(data.Url)
	if err != nil {
		logrus.Errorf("failed to visit %s, error: %v", data.Url, err)
	}

	return &data
}

type articleInfo struct {
	url string
	h   *hab
}

func (p *Parser) processRoutine(ctx context.Context) {
	for {
		select {
		case val := <-p.c:
			article := p.parseArticle(val)
			article.HabType = val.h.habInfo.HabType
			p.articlesBuf = append(p.articlesBuf, article)

		case <-ctx.Done():
			return
		}
	}
}

type hab struct {
	habInfo  models.HabInfo
	interval time.Duration
	timer    *time.Timer

	articleUrlsBuf []string
	c              chan<- articleInfo
}

func newHab(habInfo models.HabInfo, c chan articleInfo) *hab {
	return &hab{
		interval: viper.GetDuration("parser.default-interval"),
		timer:    time.NewTimer(viper.GetDuration("parser.default-interval")),
		habInfo:  habInfo,

		articleUrlsBuf: make([]string, 0),
		c:              c,
	}
}

func (h *hab) parseMainPage() {
	h.fillArticlesBuf()
	h.sendArticlesFromBufToParse()
}

func (h *hab) fillArticlesBuf() {
	collector := colly.NewCollector()

	collector.OnHTML(h.habInfo.MainPageQueryArticle, func(htmlElement *colly.HTMLElement) {
		articleUrl, ok := htmlElement.DOM.Attr("href")
		if !ok {
			panic("wrong attribute was given to selection")
		}
		h.articleUrlsBuf = append(h.articleUrlsBuf, strings.TrimSpace(h.habInfo.ArticleUrlPrefix+articleUrl))
	})

	err := collector.Visit(h.habInfo.MainUrl)
	if err != nil {
		logrus.Errorf("failed to visit url, URL: %s, error: %v", h.habInfo.MainUrl, err)
	}
}

func (h *hab) sendArticlesFromBufToParse() {
	for _, elem := range h.articleUrlsBuf {
		h.c <- articleInfo{
			url: elem,
			h:   h,
		}
	}

	h.articleUrlsBuf = h.articleUrlsBuf[:0]
}

func (p *Parser) putArticleInTable() error {
	for _, article := range p.articlesBuf {
		if article.Url == "" {
			logrus.Error(ErrUrlIsEmpty)
			continue
		}

		if article.Title == "" {
			logrus.Error(ErrTitleIsEmpty)
			continue
		}

		if article.Username == "" {
			logrus.Error(ErrUsernameIsEmpty)
			continue
		}

		if article.UsernameUrl == "" {
			logrus.Error(ErrUsernameUrlIsEmpty)
			continue
		}

		if article.PublishData == "" {
			logrus.Error(ErrDateIsEmpty)
			continue
		}

		date, err := time.Parse(time.RFC3339, article.PublishData)
		if err != nil {
			logrus.Errorf("failed to convert data to time, error: %v", err)
			continue
		}

		username := strings.TrimSpace(article.Username)
		usernameUrl := strings.TrimSpace(article.UsernameUrl)
		url := strings.TrimSpace(article.Url)

		_, err = p.storage.Put(url, username, usernameUrl, article.Title, date, article.HabType)
		if err != nil {
			logrus.Errorf("failed to put data, error: %v", err)
		}
	}

	p.articlesBuf = p.articlesBuf[:0]

	return nil
}
