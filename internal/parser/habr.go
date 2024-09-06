package parser

import (
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"testTask/internal/database"
	"time"
)

type habrParser struct {
	mainUrl                  string
	articleUrlPrefix         string
	mainPageQueryArticle     string
	articlePageQueryTitle    string
	articlePageQueryTime     string
	articlePageQueryUserLink string

	storage          *database.Database
	buf              []*ArticleData
	articleUrlsBuf   []string
	goroutinesAmount int
	c                chan string
}

func newHabrParser(db *database.Database) *habrParser {
	return &habrParser{
		mainUrl:                  viper.GetString("parser.habr.links.main-url"),
		articleUrlPrefix:         viper.GetString("parser.habr.links.article-url-prefix"),
		mainPageQueryArticle:     viper.GetString("parser.habr.main-page-query.article"),
		articlePageQueryTime:     viper.GetString("parser.habr.article-page-query.time"),
		articlePageQueryTitle:    viper.GetString("parser.habr.article-page-query.title"),
		articlePageQueryUserLink: viper.GetString("parser.habr.article-page-query.user-link"),
		articleUrlsBuf:           make([]string, 0),
		c:                        make(chan string),
		storage:                  db,
		goroutinesAmount:         viper.GetInt("parser.goroutines-amount"),
	}
}

func (h *habrParser) getHabrArticleUrlFromMainPage() {
	collector := colly.NewCollector()

	collector.OnHTML(h.mainPageQueryArticle, func(htmlElement *colly.HTMLElement) {
		articleUrl, ok := htmlElement.DOM.Attr("href")
		if !ok {
			panic("wrong attribute was given to selection")
		}
		h.articleUrlsBuf = append(h.articleUrlsBuf, strings.TrimSpace(h.articleUrlPrefix+articleUrl))
	})

	err := collector.Visit(h.mainUrl)
	if err != nil {
		logrus.Errorf("failed to visit url, URL: %s, error: %v", h.mainUrl, err)
	}
}

func (h *habrParser) parseHabrArticle(articleUrl string) *ArticleData {
	collector := colly.NewCollector()

	var data ArticleData

	data.Url = articleUrl

	collector.OnHTML(h.articlePageQueryUserLink, func(htmlElement *colly.HTMLElement) {
		data.UsernameUrl = htmlElement.Attr("href")
		data.Username = htmlElement.Text
	})

	collector.OnHTML(h.articlePageQueryTitle, func(htmlElement *colly.HTMLElement) {
		data.Title = htmlElement.Text
	})

	collector.OnHTML(h.articlePageQueryTime, func(htmlElement *colly.HTMLElement) {
		data.PublishData, _ = htmlElement.DOM.Children().Attr("datetime")
	})

	err := collector.Visit(data.Url)
	if err != nil {
		logrus.Errorf("failed to visit %s, error: %v", data.Url, err)
	}

	return &data
}

func (h *habrParser) parse() {
	for i := 0; i < h.goroutinesAmount; i++ {
		go h.processRoutine()
	}
}

func (h *habrParser) processRoutine() {
	for {
		val := <-h.c
		article := h.parseHabrArticle(val)
		err := putArticleInTable(article, h.storage)
		if err != nil {
			logrus.Errorf("failed to put data in database, error: %v", err)
		}
	}
}

func (h *habrParser) parseMainPage() {
	h.getHabrArticleUrlFromMainPage()
	h.buf = make([]*ArticleData, 0, len(h.articleUrlsBuf))
	h.sendArticlesFromBuf()
}

func (h *habrParser) sendArticlesFromBuf() {
	for _, elem := range h.articleUrlsBuf {
		h.c <- elem
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
