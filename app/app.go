package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unkeep/alfabooker/api"
	"github.com/unkeep/alfabooker/budget"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
)

type App struct{}

func (app *App) Run(ctx context.Context) error {
	log.Println("Run")
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("getConfig: %w", err)
	}

	// herocu param
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Println("GetRepo")
	repo, err := db.GetRepo(ctx, cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("db.GetRepo: %w", err)
	}

	log.Println("GetBudgetDomain")
	budgetDomain := budget.NewDomain(repo.Budget)

	httpServer := api.NewServer(port, budgetDomain, cfg.APIAuthToken)
	go httpServer.ListenAndServe()
	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Println("GetBot")
	tgBot, err := tg.GetBot(cfg.TgToken)
	if err != nil {
		return fmt.Errorf("tg.GetBot: %w", err)
	}

	msgChan := make(chan tg.UserMsg, 0)
	critErrosChan := make(chan error, 0)

	go func() {
		if err := tgBot.GetUpdates(ctx, msgChan); err != nil {
			critErrosChan <- fmt.Errorf("tgBot.GetUpdates: %w", err)
		}
	}()

	c := controller{
		cfg:          cfg,
		repo:         repo,
		tgBot:        tgBot,
		budgetDomain: budgetDomain,
	}

	cc := func(name string, param interface{}, f func(ctx context.Context) error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		if err := f(ctx); err != nil {
			log.Printf("%s(%+v): %s\n", name, param, err.Error())
			_, _ = tgBot.SendMessage(tg.BotMessage{
				ChatID: cfg.TgAdminChatID,
				Text: fmt.Sprintf("⚠️ controller: %s, error:\n```%s```\ncontext:\n```%+v```\n",
					name, err.Error(), param),
				TextMarkdown: true,
			})
		}
	}

	log.Println("selecting channels")
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-critErrosChan:
			return err
		case msg := <-msgChan:
			cc("handleUserMessage", msg, func(ctx context.Context) error {
				return c.handleUserMessage(ctx, msg)
			})
		}
	}
}
