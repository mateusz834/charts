package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mateusz834/charts/app"
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
	c, err := LoadConfig("config.json")
	if err != nil {
		return err
	}

	db, err := storage.NewSqliteStorage("./db.db")
	if err != nil {
		return err
	}

	sessionService := service.NewSessionService(&db)
	sharesService := service.NewSharesService(&db)

	a := app.NewApplication(app.OAuth{
		TokenURL:     "https://github.com/login/oauth/access_token",
		ClientID:     "14e6190e978637376f67",
		ClientSecret: c.ClientSecret,
	}, &sessionService, &sharesService)

	return a.Start()
}

type Config struct {
	ClientSecret string
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err := json.NewDecoder(f).Decode(c); err != nil {
		return nil, err
	}
	return c, nil
}
