package parser

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sync"
	"testTask/internal/database"
	"testTask/internal/models"
	"time"
)

var (
	ErrUrlIsEmpty          = errors.New("url is empty")
	ErrTitleIsEmpty        = errors.New("title is empty")
	ErrUsernameIsEmpty     = errors.New("username is empty")
	ErrUsernameUrlIsEmpty  = errors.New("usernameUrl is empty")
	ErrHabIsEmpty          = errors.New("habType is empty")
	ErrHabIsNotExist       = errors.New("such hab does not exist")
	ErrHabIsAlreadyParsing = errors.New("hab is already parsing")
)

type Parser struct {
	habs        map[string]*hab
	articlesBuf *articlesBuf
	storage     *database.Database

	ctx              context.Context
	stop             context.CancelFunc
	goroutinesAmount int
	c                <-chan articleInfo
}

// NewParser inits new Parser object
func NewParser(db *database.Database) (*Parser, error) {
	c := make(chan articleInfo)

	habs := make(map[string]*hab)
	for habType, f := range habsMap {
		habs[habType] = newHab(habType, f, c)
	}

	ctx := context.Background()
	ctx, stop := context.WithCancel(ctx)

	p := &Parser{
		articlesBuf: &articlesBuf{
			buf: make([]*models.ArticleData, 0),
			mx:  sync.Mutex{},
		},
		habs:             habs,
		storage:          db,
		goroutinesAmount: viper.GetInt("parser.goroutines-amount"),
		c:                c,
		ctx:              ctx,
		stop:             stop,
	}

	go func() {
		for {
			time.Sleep(viper.GetDuration("parser.load-data-interval"))
			logrus.Info("start put data in table")
			_ = p.putArticleInTable()
		}
	}()

	return p, nil
}

// Parse starts parsing habs from habsMap.
// It allocates new routine for every hab to parse it`s main page.
// Also, Parse setups routines for processing routines parsing.
func (p *Parser) Parse() {
	for _, h := range p.habs {
		h.setupRoutine()
	}

	for i := 0; i < p.goroutinesAmount; i++ {
		go p.processRoutine(p.ctx)
	}
}

// StopParsingHab stops timer of main page parser.
// To use this method you should specify habType of the routine, that you want to stop.
// If habType is not located in habsMap, StopParsingHab returns an error.
func (p *Parser) StopParsingHab(habType string) error {
	h, ok := p.habs[habType]
	if !ok {
		return ErrHabIsNotExist
	}

	h.timer.Stop()
	return nil
}

// AddHabForParsing method let routine resume parsing habType, who previously was stopped.
// If habType is already parsing, AddHabForParsing returns an error.
// If habType is not exist in habsMap, it also returns an error
func (p *Parser) AddHabForParsing(habType string) error {
	h, ok := p.habs[habType]
	if !ok {
		return ErrHabIsNotExist
	}

	if !h.timer.Stop() {
		h.timer.Reset(h.interval)
		return nil
	}

	return ErrHabIsAlreadyParsing
}

// ChangeIntervalForHab is used to change parse interval for current hab.
// If habType is not exist in habsMap, it returns an error.
func (p *Parser) ChangeIntervalForHab(habType string, interval string) error {
	_, ok := p.habs[habType]
	if !ok {
		return ErrHabIsNotExist
	}

	t, err := time.ParseDuration(interval)
	if err != nil {
		return err
	}

	p.habs[habType].changeParseInterval(t)
	return nil
}

// DeleteHab is used to delete hab from parsing.
// WARNING! DeleteHab deletes hab from parsing forever and also delete all information about hab from storage.
// To stop parsing hab for some time you should use StopParsingHab.
func (p *Parser) DeleteHab(habType string) ([]int, error) {
	h, ok := p.habs[habType]
	if !ok {
		return nil, ErrHabIsNotExist
	}

	h.stop()
	ids, err := p.storage.DeleteHab(habType)
	if err != nil {
		return nil, err
	}

	delete(p.habs, habType)

	return ids, nil
}

type articleInfo struct {
	url     string
	habType string
}

func (p *Parser) processRoutine(ctx context.Context) {
	for {
		select {
		case val := <-p.c:
			article := habsMap[val.habType].parseArticlePage(val.url)
			p.articlesBuf.appendBuf(article)

		case <-ctx.Done():
			return
		}
	}
}

func (p *Parser) putArticleInTable() error {
	p.articlesBuf.mx.Lock()

	for _, article := range p.articlesBuf.buf {
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

		if article.HabType == "" {
			logrus.Error(ErrHabIsEmpty)
			continue
		}

		_, err := p.storage.PutArticle(article.Url, article.Username, article.UsernameUrl, article.Title, article.PublishData, article.HabType)
		if err != nil {
			logrus.Errorf("failed to put data, error: %v", err)
		}
	}

	p.articlesBuf.buf = p.articlesBuf.buf[:0]
	p.articlesBuf.mx.Unlock()

	return nil
}

type articlesBuf struct {
	buf []*models.ArticleData
	mx  sync.Mutex
}

func (a *articlesBuf) appendBuf(data *models.ArticleData) {
	a.mx.Lock()
	a.buf = append(a.buf, data)
	a.mx.Unlock()
}
