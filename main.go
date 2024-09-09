package main

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"html/template"
	"os"
	"os/signal"
	"testTask/internal/cast"
	"testTask/internal/database"
	"testTask/internal/endpoint"
	"testTask/internal/parser"
)

var (
	pars    *parser.Parser
	db      *database.Database
	handler *endpoint.HttpHandler
)

func main() {
	setupConfig()
	setupDatabase()
	setupParser()
	setupHttpHandler()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}

func setupHttpHandler() {
	handler = endpoint.NewHttpHandler(pars)
	go func() {
		logrus.Info("Server started")
		err := fasthttp.ListenAndServe(viper.GetString("server.host")+":"+viper.GetString("server.port"), handler.Handle)
		if err != nil {
			logrus.Fatal("Listen error: ", err.Error())
		}
	}()
}

func setupDatabase() {
	var err error
	db, err = database.NewDatabase()
	if err != nil {
		logrus.Fatalf("failed to setup database, error: %v", err)
	}
}

func setupParser() {
	var err error
	pars, err = parser.NewParser(db)
	if err != nil {
		logrus.Fatalf("failed to setup parser")
	}

	pars.Parse()
}

func setupConfig() {
	viper.SetConfigFile("./configuration.yaml")

	file, err := os.ReadFile("./configuration.yaml")
	if err != nil {
		logrus.Fatalf("failed to read configuration file, error: %v", err)
	}

	tmpl := template.New("config")
	tmpl.Funcs(template.FuncMap{
		"env": func(name string) string {
			return os.Getenv(name)
		},
	})

	tmpl, err = tmpl.Parse(cast.ByteArrayToSting(file))
	if err != nil {
		logrus.Fatalf("failed to parse template, error: %v", err)
	}

	var configData bytes.Buffer
	err = tmpl.Execute(&configData, nil)
	if err != nil {
		logrus.Fatalf("failed to execute template, error: %v", err)
	}

	err = viper.ReadConfig(&configData)
	if err != nil {
		panic(err)
	}
}
