package database

import (
	"context"
	"database/sql"
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
	getHabInfo                 *pgconn.StatementDescription
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

	getHabInfo, err := conn.Prepare(context.Background(), "Get hab", `SELECT habType FROM habs WHERE habType = $1`)
	if err != nil {
		logrus.Errorf("failed to prepare getHabInfoStmt, error: %v", err)
	}

	getFromHabsInformationStmt, err := conn.Prepare(context.Background(), "Get Hab Information", "SELECT * FROM habs")
	if err != nil {
		logrus.Errorf("failed to prepare getFromHabsInformationStmt, error: %v", err)
	}

	return &Database{db: conn,
		getHabInfo:                 getHabInfo,
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

func (d *Database) PutHab(habType string, mainPageUrl string) error {
	logrus.Infof("put data %s", habType)
	d.db.QueryRow(context.Background(), d.putInformationInHabsStmt.Name, habType, mainPageUrl)
	return nil
}

func (d *Database) GetHabInfo(habType string) error {
	var str string
	err := d.db.QueryRow(context.Background(), d.getHabInfo.Name, habType).Scan(&str)
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
