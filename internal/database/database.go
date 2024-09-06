package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Database struct {
	db                  *pgx.Conn
	putInArticlesStmt   *pgconn.StatementDescription
	getFromTableStmt    *pgconn.StatementDescription
	deleteFromTableStmt *pgconn.StatementDescription
	getAllFromTableStmt *pgconn.StatementDescription
}

var (
	ErrRowNotExist = errors.New("row with such id do not exist")
)

func NewDatabase() (*Database, error) {
	username := viper.GetString("database.username")
	password := os.Getenv(viper.GetString("database.password"))
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

	putInArticlesStmt, err := conn.Prepare(context.Background(), "Put", `INSERT INTO articles(articleURL, username, usernameURL, title, date) VALUES ($1, $2, $3, $4, $5) RETURNING id`)
	if err != nil {
		logrus.Errorf("failed to prepare createTableStmt, error: %v", err)
		return nil, err
	}

	getFromTableStmt, err := conn.Prepare(context.Background(), "GetById", `SELECT json_data FROM products WHERE id = $1`)
	if err != nil {
		logrus.Errorf("failed to prepare getFromTableStmt, error: %v", err)
		return nil, err
	}

	deleteFromTableStmt, err := conn.Prepare(context.Background(), "DeleteById", `DELETE FROM products where id = $1`)
	if err != nil {
		logrus.Errorf("failed to prepare deleteFromTableStmt, error: %v", err)
		return nil, err
	}

	getAllFromTable, err := conn.Prepare(context.Background(), "GetAllFromDb", `SELECT id, json_data FROM products`)
	if err != nil {
		logrus.Errorf("failed to prepare getAllFromTableStmt, error: %v", err)
		return nil, err
	}

	return &Database{db: conn,
		putInArticlesStmt:   putInArticlesStmt,
		getFromTableStmt:    getFromTableStmt,
		deleteFromTableStmt: deleteFromTableStmt,
		getAllFromTableStmt: getAllFromTable,
	}, nil
}

func (d *Database) Put(articleUrl string, username string, usernameUrl string, title string, date time.Time) (int, error) {
	var id int
	if err := d.db.QueryRow(context.Background(), d.putInArticlesStmt.Name, articleUrl, username, usernameUrl, title, date).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}
