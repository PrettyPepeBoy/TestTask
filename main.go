package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"testTask/internal/parser"
)

var (
	pars *parser.Parser
)

func main() {
	setupConfig()
	setupParser()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}

func setupParser() {
	pars = parser.NewParser()
	pars.Parse()
	pars.GetNewArticles()
}

func setupConfig() {
	viper.SetConfigFile("./configuration.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("failed to read config, error: %v", err)
	}
}
