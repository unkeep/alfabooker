package ontrackfunc

import (
	"log"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	"github.com/unkeep/alfabooker/app"
)

func init() {
	h, err := app.NewHandler()
	if err != nil {
		log.Fatal("Init error: ", err.Error())
	}

	functions.HTTP("ServeHTTP", h.ServeHTTP)
}
