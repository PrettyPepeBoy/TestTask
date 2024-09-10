package models

import "time"

type ArticleData struct {
	Username    string    `json:"username"`
	UsernameUrl string    `json:"usernameUrl"`
	Title       string    `json:"title"`
	Url         string    `json:"url"`
	PublishData time.Time `json:"publishData"`
	HabType     string    `json:"habType"`
}

type HabInfo struct {
	HabType     string
	MainPageUrl string
}
