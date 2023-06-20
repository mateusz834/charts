package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mateusz834/charts/app"
	"github.com/mateusz834/charts/log"
	"github.com/mateusz834/charts/service"
	"github.com/mateusz834/charts/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	confPath := flag.String("config", "./config.json", "")
	flag.Parse()

	c, err := LoadConfig(*confPath)
	if err != nil {
		return err
	}

	db, err := storage.NewSqliteStorage(c.DB)
	if err != nil {
		return err
	}

	sessionService := service.NewSessionService(&db)
	sharesService := service.NewSharesService(&db)

	var logger log.Logger = &log.ConsoleLogger{}
	if c.Syslog {
		logger = log.NewSyslogLogger()
	}

	a := app.NewApplication(app.OAuth{
		TokenURL:     "https://github.com/login/oauth/access_token",
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
	}, logger, &sessionService, &sharesService)

	return a.Start(c.Addr)
}

type Config struct {
	ClientSecret string
	ClientID     string
	Syslog       bool
	Addr         string
	DB           string
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	c := &Config{
		Addr: "127.0.0.1:8888",
	}

	if err := json.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}
