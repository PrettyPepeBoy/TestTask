package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/sirupsen/logrus"
	"strings"
)

func main() {
	fmt.Println(parse())

}

//func parse() []string {
//	collector := colly.NewCollector()
//
//	buf := make([]string, 0)
//
//	collector.OnHTML("div.postrepeter-left", func(htmlElement *colly.HTMLElement) {
//		txt, _ := htmlElement.DOM.Children().Attr("href")
//		buf = append(buf, txt)
//	})
//
//	err := collector.Visit("https://www.techopedia.com/article/topics")
//	if err != nil {
//		logrus.Fatalf("failed to visit, error: %v", err)
//	}
//	return buf
//}

func parse() []string {
	collector := colly.NewCollector()

	buf := make([]string, 0)

	collector.OnHTML("a.card-articles__body-link", func(htmlElement *colly.HTMLElement) {
		articleUrl := htmlElement.Attr("href")

		buf = append(buf, strings.TrimSpace(articleUrl))
	})

	err := collector.Visit("https://skillbox.ru/media/topic/articles/")
	if err != nil {
		logrus.Errorf("failed to visit url, URL: %s, error: %v", "", err)
	}

	return buf

}
