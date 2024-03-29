package app

import (
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	TgToken       string `required:"true"`
	TgAdminChatID int64  `required:"true"`
	MongoURI      string `required:"true"`
	APIAuthToken  string `required:"true"`
	URL           string `required:"true"`
}

func getConfig() (config, error) {
	var cfg config
	err := envconfig.Process("AB", &cfg)

	return cfg, err
}
