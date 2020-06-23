package app

import (
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	TgToken       string `required:"true"`
	TgChatID      int64  `required:"true"`
	TgAdminChatID int64  `required:"true"`
	GSheetID      string `required:"true"`
	GClientID     string `required:"true"`
	GClientSecret string `required:"true"`
	GProjectID    string `required:"true"`
	MongoURI      string `required:"true"`
	Port          string `default:"80"`
}

func getConfig() (config, error) {
	var cfg config
	err := envconfig.Process("AB", &cfg)

	return cfg, err
}
