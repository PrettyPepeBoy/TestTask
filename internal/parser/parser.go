package parser

import (
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"testTask/internal/database"
	"time"
)

type Parser struct {
	habs    []*hab
	storage *database.Database

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

	return &Parser{
		habs:             habs,
		storage:          db,
		goroutinesAmount: viper.GetInt("parser.goroutines-amount"),
		c:                c,
	}, nil
}

func (p *Parser) Parse() {
	for _, h := range p.habs {
		go func(h *hab) {
			for {
				<-h.timer.C
				h.parseMainPage()
				h.timer.Reset(h.interval)
			}
		}(h)

	}

	p.setupParseArticle()
}

func (p *Parser) parseArticle(info articleInfo) *ArticleData {
	collector := colly.NewCollector()

	var data ArticleData

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

func (p *Parser) setupParseArticle() {
	for i := 0; i < p.goroutinesAmount; i++ {
		go p.processRoutine()
	}
}

type articleInfo struct {
	url string
	h   *hab
}

func (p *Parser) processRoutine() {
	for {
		val := <-p.c
		article := p.parseArticle(val)
		err := putArticleInTable(article, p.storage)
		if err != nil {
			logrus.Errorf("failed to put data in database, error: %v", err)
		}
	}
}

type hab struct {
	habInfo  HabInfo
	interval time.Duration
	timer    *time.Timer

	articleUrlsBuf []string
	c              chan<- articleInfo
}

func newHab(habInfo HabInfo, c chan articleInfo) *hab {
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

func putArticleInTable(article *ArticleData, db *database.Database) error {
	if article.Url == "" {
		return ErrUrlIsEmpty
	}

	if article.Title == "" {
		return ErrTitleIsEmpty
	}

	if article.Username == "" {
		return ErrUsernameIsEmpty
	}

	if article.UsernameUrl == "" {
		return ErrUsernameUrlIsEmpty
	}

	if article.PublishData == "" {
		return ErrDateIsEmpty
	}

	date, err := time.Parse(time.RFC3339, article.PublishData)
	if err != nil {
		logrus.Errorf("failed to convert data to time, error: %v", err)
		return err
	}

	username := strings.TrimSpace(article.Username)
	usernameUrl := strings.TrimSpace(article.UsernameUrl)
	url := strings.TrimSpace(article.Url)

	_, err = db.Put(url, username, usernameUrl, article.Title, date)
	if err != nil {
		return err
	}

	return nil
}
