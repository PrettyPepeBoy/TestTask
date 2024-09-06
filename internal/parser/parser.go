package parser

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

var (
	ErrUsernameUrlIsNotExist = errors.New("can not add data without usernameUrl")
)

type Parser struct {
	habr *habrParser

	interval time.Duration
	timer    *time.Timer
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
		interval: viper.GetDuration("parser.default-interval"),
		timer:    time.NewTimer(viper.GetDuration("parser.default-interval")),
		habr:     newHabrParser(),
	}
}

func (p *Parser) ParseHabrPage() {
	go func() {
		for {
			<-p.timer.C
			p.habr.parseMainPage()
			fmt.Println(p.habr.buf)
			p.timer.Reset(p.interval)
		}
	}()

	p.habr.parse()
}
