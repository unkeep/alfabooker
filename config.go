package main

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TgToken    string `required:"true"`
	TgChatID   int64  `required:"true"`
	GmailLogin string `required:"true"`
	GmailPass  string `required:"true"`
	GSheetID   string `required:"true"`
}

func GetConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("AB", &cfg)

	return cfg, err
}
