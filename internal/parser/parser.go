package parser

import (
	"errors"
	"github.com/spf13/viper"
	"time"
)

var (
	ErrUsernameUrlIsNotExist = errors.New("can not add data without usernameUrl")
)

type Parser struct {
	buf  []*ArticleData
	habr *habrParser

	interval         time.Duration
	timer            *time.Timer
	goroutinesAmount int
}

type ArticleData struct {
	Username    string
	UsernameUrl string
	Title       string
	Url         string
	PublishData time.Time
}

func NewParser() *Parser {
	return &Parser{
		goroutinesAmount: viper.GetInt("parser.goroutines-amount"),
		interval:         viper.GetDuration("parser.default-interval"),
		timer:            time.NewTimer(viper.GetDuration("parser.default-interval")),
		habr:             setupHabrParser(),
	}
}

func (p *Parser) GetNewArticles() {
	go func() {
		for {
			<-p.timer.C
			p.habr.getHabrArticleUrlFromMainPage()
			p.buf = make([]*ArticleData, 0, len(p.habr.articlesBuf))
			p.habr.processParing()
			p.timer.Reset(time.Hour)
		}
	}()
}

func (p *Parser) Parse() {
	for i := 0; i < p.goroutinesAmount; i++ {
		go p.processRoutine(p.habr.c)
	}
}

func (p *Parser) processRoutine(c chan string) {
	for {
		val := <-c
		article := p.habr.parseHabrArticle(val)
		p.buf = append(p.buf, article)
	}
}
