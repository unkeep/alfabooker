package main

import (
	"context"
	"log"
	_ "net/http/pprof"

	"github.com/unkeep/alfabooker/app"
)

func main() {
	ctx := context.Background()
	// TODO: handler int sig
	if err := new(app.App).Run(ctx); err != nil {
		log.Fatal(err)
	}
}
