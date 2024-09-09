package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"strings"
	"testTask/internal/models"
	"time"
)

func main() {
	data := parse("https://skillbox.ru/media/management/chto-takoe-kastomizatsiya-i-zachem-ona-nuzhna-biznesu-i-klientam/")
	fmt.Println(data)
}

func parse(url string) models.ArticleData {
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

	return data
}

//446d69747269693230303131303037
