package parser

import (
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"time"
)

type habrParser struct {
	mainUrl                  string
	articleUrlPrefix         string
	mainPageQueryArticle     string
	articlePageQueryTitle    string
	articlePageQueryTime     string
	articlePageQueryUserLink string

	buf              []*ArticleData
	articleUrlsBuf   []string
	goroutinesAmount int
	c                chan string
}

func newHabrParser() *habrParser {
	return &habrParser{
		mainUrl:                  viper.GetString("parser.habr.links.main-url"),
		articleUrlPrefix:         viper.GetString("parser.habr.links.article-url-prefix"),
		mainPageQueryArticle:     viper.GetString("parser.habr.main-page-query.article"),
		articlePageQueryTime:     viper.GetString("parser.habr.article-page-query.time"),
		articlePageQueryTitle:    viper.GetString("parser.habr.article-page-query.title"),
		articlePageQueryUserLink: viper.GetString("parser.habr.article-page-query.user-link"),
		articleUrlsBuf:           make([]string, 0),
		c:                        make(chan string),
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

		usernameUrl := htmlElement.Attr("href")
		username := htmlElement.Text

		data.UsernameUrl = strings.TrimSpace(usernameUrl)
		data.Username = strings.TrimSpace(username)
	})

	collector.OnHTML(h.articlePageQueryTitle, func(htmlElement *colly.HTMLElement) {
		data.Title = htmlElement.Text
	})

	collector.OnHTML(h.articlePageQueryTime, func(htmlElement *colly.HTMLElement) {
		dateString, _ := htmlElement.DOM.Children().Attr("datetime")

		var err error
		data.PublishData, err = time.Parse(time.RFC3339, dateString)
		if err != nil {
			logrus.Errorf("failed to convert data to time, error: %v", err)
		}
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
		h.buf = append(h.buf, article)
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
