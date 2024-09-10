package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testTask/internal/models"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Database struct {
	mx                         sync.Mutex
	db                         *pgx.Conn
	getArticlesStmt            *pgconn.StatementDescription
	putInArticlesStmt          *pgconn.StatementDescription
	putInformationInHabsStmt   *pgconn.StatementDescription
	getFromHabsInformationStmt *pgconn.StatementDescription
	getHabInfoStmt             *pgconn.StatementDescription
	deleteHabStmt              *pgconn.StatementDescription
	deleteArticlesStmt         *pgconn.StatementDescription
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
	CREATE TABLE IF NOT EXISTS articles (id serial, articleUrl  text, username text, usernameUrl text, title text, date time, habType text references habs(habType));`)

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

	getHabInfoStmt, err := conn.Prepare(context.Background(), "Get hab", `SELECT habType FROM habs WHERE habType = $1`)
	if err != nil {
		logrus.Errorf("failed to prepare getHabInfoStmt, error: %v", err)
	}

	getFromHabsInformationStmt, err := conn.Prepare(context.Background(), "Get Hab Information", "SELECT * FROM habs")
	if err != nil {
		logrus.Errorf("failed to prepare getFromHabsInformationStmt, error: %v", err)
	}

	deleteHabStmt, err := conn.Prepare(context.Background(), "Delete hab", "DELETE FROM habs WHERE habType = $1 RETURNING habType")
	if err != nil {
		logrus.Errorf("failed to prepare deleteHabStmt, error: %v", err)
	}

	deleteArticlesStmt, err := conn.Prepare(context.Background(), "Delete Articles", `DELETE FROM articles WHERE habType = $1 RETURNING id`)
	if err != nil {
		logrus.Errorf("failed to prepare deleteArticlesStmt, error: %v", err)
	}

	getArticlesStmt, err := conn.Prepare(context.Background(), "Get Articles", `Select * FROM articles`)
	if err != nil {
		logrus.Errorf("failed to prepare getArticlesStmt, error: %v", err)
	}

	return &Database{db: conn,
		getArticlesStmt:            getArticlesStmt,
		getHabInfoStmt:             getHabInfoStmt,
		putInArticlesStmt:          putInArticlesStmt,
		putInformationInHabsStmt:   putInformationInHabsStmt,
		getFromHabsInformationStmt: getFromHabsInformationStmt,
		deleteHabStmt:              deleteHabStmt,
		deleteArticlesStmt:         deleteArticlesStmt,
		mx:                         sync.Mutex{},
	}, nil
}

func (d *Database) PutArticle(articleUrl string, username string, usernameUrl string, title string, date time.Time, habType string) (int, error) {
	var id int
	if err := d.db.QueryRow(context.Background(), d.putInArticlesStmt.Name, articleUrl, username, usernameUrl, title, date, habType).Scan(&id); err != nil {

		return 0, err
	}

	return id, nil
}

func (d *Database) GetArticles() ([]models.ArticleData, error) {
	d.mx.Lock()
	defer d.mx.Unlock()

	rows, err := d.db.Query(context.Background(), d.getArticlesStmt.Name)
	if err != nil {
		logrus.Errorf("failed to get data from database, error: %v", err)
		return nil, err
	}

	articles := make([]models.ArticleData, 0)

	for rows.Next() {
		var id int
		var article models.ArticleData
		err = rows.Scan(&id, &article.Url, &article.Username, &article.UsernameUrl, &article.Title, &article.PublishData, &article.HabType)
		if err != nil {
			logrus.Errorf("failed to scan data, error: %v", err)
			continue
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (d *Database) PutHab(habType string, mainPageUrl string) error {
	logrus.Infof("put data %s", habType)
	d.db.QueryRow(context.Background(), d.putInformationInHabsStmt.Name, habType, mainPageUrl)
	return nil
}

func (d *Database) GetHabInfo(habType string) error {
	var str string
	err := d.db.QueryRow(context.Background(), d.getHabInfoStmt.Name, habType).Scan(&str)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRowNotExist
		}

		return err
	}

	return nil
}

func (d *Database) GetHabsInfo() ([]models.HabInfo, error) {
	rows, err := d.db.Query(context.Background(), d.getFromHabsInformationStmt.Name)
	if err != nil {
		logrus.Errorf("failed to put data in table, error: %v", err)
		return nil, err
	}

	var (
		habType string
		mainUrl string
	)
	habInfo := make([]models.HabInfo, 0)

	for rows.Next() {
		err = rows.Scan(&habType, &mainUrl)
		if err != nil {
			logrus.Errorf("failed to scan data in %s, error: %v", habType, err)
			continue
		}

		habInfo = append(habInfo, models.HabInfo{
			HabType:     habType,
			MainPageUrl: mainUrl,
		})
	}

	return habInfo, nil
}

func (d *Database) DeleteHab(habType string) ([]int, error) {
	d.mx.Lock()
	defer d.mx.Unlock()

	var ids []int

	tx, err := d.db.Begin(context.Background())
	if err != nil {
		logrus.Errorf("failed to init transaction, error: %v", err)
		return nil, err
	}

	rows, err := tx.Query(context.Background(), d.deleteArticlesStmt.Name, habType)
	if err != nil {
		tx.Rollback(context.Background())
		return nil, err
	}

	var id int
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			logrus.Errorf("failed to scan id, error: %v", err)
			continue
		}

		ids = append(ids, id)
	}

	var hab string
	err = tx.QueryRow(context.Background(), d.deleteHabStmt.Name, habType).Scan(&hab)
	if err != nil {
		logrus.Errorf("failed to scan to hab, error: %v", err)
		tx.Rollback(context.Background())
		return nil, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		logrus.Errorf("failed to commit transaction, error: %v", err)
		return nil, err
	}
	return ids, nil
}
