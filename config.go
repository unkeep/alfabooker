package main

import (
	"github.com/kelseyhightower/envconfig"
)

// Config is an application config
type Config struct {
	TgToken       string `required:"true"`
	TgChatID      int64  `required:"true"`
	GSheetID      string `required:"true"`
	GClientID     string `required:"true"`
	GClientSecret string `required:"true"`
	GProjectID    string `required:"true"`
	MongoURI      string `required:"true"`
}

// GetConfig gets a config from env vars
func GetConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("AB", &cfg)

	return cfg, err
}
