package models

type ArticleData struct {
	Username    string
	UsernameUrl string
	Title       string
	Url         string
	PublishData string
	HabType     string
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
