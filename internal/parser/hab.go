package parser

import (
	"context"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"testTask/internal/cast"
	"testTask/internal/models"
	"time"
)

var habsMap = map[string]habParseFunctions{
	"habr": {
		parseMainPage: func(buf []string) []string {
			collector := colly.NewCollector()

			collector.OnHTML("a.tm-title__link", func(htmlElement *colly.HTMLElement) {
				articleUrl, ok := htmlElement.DOM.Attr("href")
				if !ok {
					panic("wrong attribute was given to selection")
				}
				buf = append(buf, strings.TrimSpace("https://habr.com"+articleUrl))
			})

			err := collector.Visit("https://habr.com/ru/articles/")
			if err != nil {
				logrus.Errorf("failed to visit url, URL: %s, error: %v", "https://habr.com/ru/articles/", err)
			}

			return buf
		},

		parseArticlePage: func(url string) *models.ArticleData {
			collector := colly.NewCollector()

			var data models.ArticleData
			data.Url = url

			collector.OnHTML("a.tm-user-info__username", func(htmlElement *colly.HTMLElement) {
				data.UsernameUrl = "https://habr.com" + htmlElement.Attr("href")
				data.Username = htmlElement.Text
			})

			collector.OnHTML("h1.tm-title", func(htmlElement *colly.HTMLElement) {
				data.Title = htmlElement.Text
			})

			collector.OnHTML("span.tm-article-datetime-published", func(htmlElement *colly.HTMLElement) {
				publishData, _ := htmlElement.DOM.Children().Attr("title")
				slc := cast.StringToByteArray(publishData)
				slc = append(slc[:10], slc[11:]...)
				dt := publishData[:(len(publishData)-1)] + ":00"

				var err error
				data.PublishData, err = time.Parse(time.DateTime, dt)
				if err != nil {
					panic(err)
				}
			})

			data.HabType = "habr"

			err := collector.Visit(url)
			if err != nil {
				logrus.Errorf("failed to visit url, URL: %s, error: %v", url, err)
			}

			return &data
		},

		habMainPageUrl: "https://habr.com/ru/articles/",
	},

	"skillbox": {
		parseMainPage: func(buf []string) []string {
			collector := colly.NewCollector()

			collector.OnHTML("a.card-articles__body-link", func(htmlElement *colly.HTMLElement) {
				articleUrl := htmlElement.Attr("href")

				buf = append(buf, strings.TrimSpace("https://skillbox.ru"+articleUrl))
			})

			err := collector.Visit("https://skillbox.ru/media/topic/articles/")
			if err != nil {
				logrus.Errorf("failed to visit url, URL: %s, error: %v", "https://skillbox.ru/media/topic/articles/", err)
			}

			return buf
		},

		parseArticlePage: func(url string) *models.ArticleData {
			collector := colly.NewCollector()

			var data models.ArticleData
			data.Url = url

			collector.OnHTML("div.article-author__name", func(htmlElement *colly.HTMLElement) {
				data.Username = strings.TrimSpace(htmlElement.Text)
			})

			collector.OnHTML("div.article-author__image", func(htmlElement *colly.HTMLElement) {
				data.UsernameUrl, _ = htmlElement.DOM.Children().Attr("href")
			})

			collector.OnHTML("h1.article-preview__title", func(htmlElement *colly.HTMLElement) {
				data.Title = strings.TrimSpace(htmlElement.Text)
			})

			collector.OnHTML("time.info-text", func(htmlElement *colly.HTMLElement) {
				data.PublishData = time.Now()
			})

			data.HabType = "skillbox"

			err := collector.Visit(url)
			if err != nil {
				logrus.Errorf("failed to visit url, URL: %s, error: %v", "https://skillbox", err)
			}

			return &data
		},

		habMainPageUrl: "https://skillbox.ru/media/topic/articles/",
	},
}

type hab struct {
	habType        string
	parseFunctions habParseFunctions
	interval       time.Duration
	timer          *time.Timer
	usedArticles   map[string]struct{}
	articleUrlsBuf []string
	c              chan<- articleInfo
	ctx            context.Context
	stop           context.CancelFunc
}

type habParseFunctions struct {
	parseMainPage    func(buf []string) []string
	parseArticlePage func(url string) *models.ArticleData
	habMainPageUrl   string
}

func newHab(habType string, f habParseFunctions, c chan articleInfo) *hab {
	ctx := context.Background()
	ctx, stop := context.WithCancel(ctx)

	return &hab{
		habType:        habType,
		parseFunctions: f,
		interval:       viper.GetDuration("parser.default-interval"),
		timer:          time.NewTimer(viper.GetDuration("parser.default-interval")),
		usedArticles:   make(map[string]struct{}),
		articleUrlsBuf: make([]string, 0),
		c:              c,
		ctx:            ctx,
		stop:           stop,
	}
}

func (h *hab) parseMainPage() {
	h.fillArticlesBuf()
	h.sendArticlesFromBufToParse()
}

func (h *hab) fillArticlesBuf() {
	logrus.Infof("statt fill articles buf on %s, timer: %v", h.habType, h.interval)
	h.articleUrlsBuf = h.parseFunctions.parseMainPage(h.articleUrlsBuf)
}

func (h *hab) sendArticlesFromBufToParse() {
	for _, elem := range h.articleUrlsBuf {
		if _, ok := h.usedArticles[elem]; ok {
			continue
		}

		h.c <- articleInfo{
			url:     elem,
			habType: h.habType,
		}

		h.usedArticles[elem] = struct{}{}
	}

	h.articleUrlsBuf = h.articleUrlsBuf[:0]
}

func (h *hab) setupRoutine() {
	go func() {
		for {
			select {
			case <-h.timer.C:
				h.parseMainPage()
				h.timer.Reset(h.interval)

			case <-h.ctx.Done():
				return
			}
		}
	}()
}

func (h *hab) stopRoutine() {
	h.stop()
}

func (h *hab) changeParseInterval(interval time.Duration) {
	h.interval = interval
}
