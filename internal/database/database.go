package database

import (
	"context"
	"errors"
	"fmt"
	"testTask/internal/models"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Database struct {
	db                         *pgx.Conn
	putInArticlesStmt          *pgconn.StatementDescription
	putInformationInHabsStmt   *pgconn.StatementDescription
	getFromHabsInformationStmt *pgconn.StatementDescription
	getFromTableStmt           *pgconn.StatementDescription
	deleteFromTableStmt        *pgconn.StatementDescription
	getAllFromTableStmt        *pgconn.StatementDescription
}

var (
	ErrRowNotExist = errors.New("row with such id do not exist")
)

func NewDatabase() (*Database, error) {
	username := viper.GetString("database.username")
	password := viper.GetString("database.password")
	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")
	database := viper.GetString("database.database")

	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", username, password, host, port, database))
	if err != nil {
		return nil, err
	}

	if err = conn.Ping(context.Background()); err != nil {
		return nil, err
	}

	_, err = conn.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS habs(habType text unique, habMainPageUrl text unique);
	CREATE TABLE IF NOT EXISTS articles (id serial, articleUrl  text, username text, usernameUrl text, title text, date time, habType text references habs (habType));`)

	putInArticlesStmt, err := conn.Prepare(context.Background(), "Put Article", `INSERT INTO articles(articleURL, username, usernameURL, title, date, habType) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`)
	if err != nil {
		logrus.Errorf("failed to prepare putInAriclesStmt, error: %v", err)
		return nil, err
	}

	putInformationInHabsStmt, err := conn.Prepare(context.Background(), "Put habs", `INSERT INTO habs(habType, habMainPageUrl) VALUES ($1, $2)`)
	if err != nil {
		logrus.Errorf("failed to preapre putInformationInHabsStmt, error: %v", err)
		return nil, err
	}

	getFromHabsInformationStmt, err := conn.Prepare(context.Background(), "Get Hab Information", "SELECT * FROM habs")
	if err != nil {
		logrus.Errorf("failed to prepare getFromHabsInformationStmt, error: %v", err)
	}

	//getFromTableStmt, err := conn.Prepare(context.Background(), "GetById", `SELECT json_data FROM products WHERE id = $1`)
	//if err != nil {
	//	logrus.Errorf("failed to prepare getFromTableStmt, error: %v", err)
	//	return nil, err
	//}
	//
	//deleteFromTableStmt, err := conn.Prepare(context.Background(), "DeleteById", `DELETE FROM products where id = $1`)
	//if err != nil {
	//	logrus.Errorf("failed to prepare deleteFromTableStmt, error: %v", err)
	//	return nil, err
	//}
	//
	//getAllFromTable, err := conn.Prepare(context.Background(), "GetAllFromDb", `SELECT id, json_data FROM products`)
	//if err != nil {
	//	logrus.Errorf("failed to prepare getAllFromTableStmt, error: %v", err)
	//	return nil, err
	//}

	return &Database{db: conn,
		putInArticlesStmt:          putInArticlesStmt,
		putInformationInHabsStmt:   putInformationInHabsStmt,
		getFromHabsInformationStmt: getFromHabsInformationStmt,
	}, nil
}

func (d *Database) PutArticle(articleUrl string, username string, usernameUrl string, title string, date time.Time, habType string) (int, error) {
	var id int
	if err := d.db.QueryRow(context.Background(), d.putInArticlesStmt.Name, articleUrl, username, usernameUrl, title, date, habType).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (d *Database) setHabsInfo() error {
	rows, err := d.db.Query(context.Background(), d.getFromHabsInformationStmt.Name)
	if err != nil {
		logrus.Errorf("failed to put data in table, error: %v", err)
		return err
	}

	var (
		habType                  string
		mainUrl                  string
		mainPageQueryArticle     string
		articleUrlPrefix         string
		articlePageQueryUserLink string
		articlePageQueryTitle    string
		articlePageQueryTime     string
	)
	habInfo := make([]models.HabInfo, 0)

	for rows.Next() {
		err = rows.Scan(&habType, &mainUrl, &mainPageQueryArticle, &articleUrlPrefix, &articlePageQueryUserLink, &articlePageQueryTitle, &articlePageQueryTime)
		if err != nil {
			logrus.Errorf("failed to scan data in %s, error: %v", habType, err)
			continue
		}

		habInfo = append(habInfo, models.HabInfo{
			HabType:                  habType,
			MainUrl:                  mainUrl,
			ArticleUrlPrefix:         mainPageQueryArticle,
			MainPageQueryArticle:     articleUrlPrefix,
			ArticlePageQueryTitle:    articlePageQueryUserLink,
			ArticlePageQueryTime:     articlePageQueryTitle,
			ArticlePageQueryUserLink: articlePageQueryTime,
		})
	}

	return habInfo, nil
}
