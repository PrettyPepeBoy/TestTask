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
	ErrUrlIsEmpty         = errors.New("url is empty")
	ErrTitleIsEmpty       = errors.New("title is empty")
	ErrUsernameIsEmpty    = errors.New("username is empty")
	ErrUsernameUrlIsEmpty = errors.New("usernameUrl is empty")
	ErrHabIsEmpty         = errors.New("habType is empty")
)

type Parser struct {
	habs        []*hab
	articlesBuf *articlesBuf
	storage     *database.Database

	ctx              context.Context
	stop             context.CancelFunc
	goroutinesAmount int
	c                <-chan articleInfo
}

func NewParser(db *database.Database) (*Parser, error) {
	c := make(chan articleInfo)

	habs := make([]*hab, 0, len(habsMap))
	for habType, f := range habsMap {
		habs = append(habs, newHab(habType, f, c))
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
			time.Sleep(20 * time.Second)
			logrus.Info("start put data in table")
			_ = p.putArticleInTable()
		}
	}()

	return p, nil
}

func (p *Parser) Parse() {
	for _, h := range p.habs {
		h.setupRoutine()
	}

	for i := 0; i < p.goroutinesAmount; i++ {
		go p.processRoutine(p.ctx)
	}
}

func (p *Parser) AddHub() {

}

func (p *Parser) UnsafeStopParse() {
	p.stop()
}

type articleInfo struct {
	url     string
	habType string
}

func (p *Parser) processRoutine(ctx context.Context) {
	for {
		select {
		case parse := <-p.c:
			article := habsMap[parse.habType].parseArticlePage(parse.url)
			logrus.Infof("parsed article on url %s", parse.url)
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

		logrus.Infof("Put Aritcle %s", article.Title)
		_, err := p.storage.Put(article.Url, article.Username, article.UsernameUrl, article.Title, article.PublishData, article.HabType)
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
