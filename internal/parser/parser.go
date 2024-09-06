package parser

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"testTask/internal/database"
	"time"
)

var (
	ErrUsernameUrlIsEmpty = errors.New("username url is empty")
	ErrUsernameIsEmpty    = errors.New("username is empty")
	ErrTitleIsEmpty       = errors.New("title is empty")
	ErrUrlIsEmpty         = errors.New("url is empty")
	ErrDateIsEmpty        = errors.New("date is empty")
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
	PublishData string
}

func NewParser(db *database.Database) *Parser {
	return &Parser{
		interval: viper.GetDuration("parser.default-interval"),
		timer:    time.NewTimer(viper.GetDuration("parser.default-interval")),
		habr:     newHabrParser(db),
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
