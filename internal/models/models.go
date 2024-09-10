package models

import "time"

type ArticleData struct {
	Username    string
	UsernameUrl string
	Title       string
	Url         string
	PublishData time.Time
	HabType     string
}

type HabInfo struct {
	HabType     string
	MainPageUrl string
}
