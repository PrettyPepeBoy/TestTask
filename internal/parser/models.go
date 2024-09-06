package parser

import "errors"

type ArticleData struct {
	Username    string
	UsernameUrl string
	Title       string
	Url         string
	PublishData string
}

type HabInfo struct {
	HabType                  string
	MainUrl                  string
	ArticleUrlPrefix         string
	MainPageQueryArticle     string
	ArticlePageQueryTitle    string
	ArticlePageQueryTime     string
	ArticlePageQueryUserLink string
}

var (
	ErrUrlIsEmpty         = errors.New("url is empty")
	ErrTitleIsEmpty       = errors.New("title is empty")
	ErrUsernameIsEmpty    = errors.New("username is empty")
	ErrUsernameUrlIsEmpty = errors.New("usernameUrl is empty")
	ErrDateIsEmpty        = errors.New("date is empty")
)
