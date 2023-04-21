package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/unkeep/alfabooker/api"
	"github.com/unkeep/alfabooker/budget"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
)

func NewHandler() (http.Handler, error) {
	ctx := context.Background()

	log.Println("Run")
	cfg, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("getConfig: %w", err)
	}

	log.Println("GetRepo")
	repo, err := db.GetRepo(ctx, cfg.MongoURI)
	if err != nil {
		return nil, fmt.Errorf("db.GetRepo: %w", err)
	}

	log.Println("GetBudgetDomain")
	budgetDomain := budget.NewDomain(repo.Budget)

	log.Println("GetBot")
	msgChan := make(chan tg.UserMsg, 0)
	tgBot, err := tg.GetBot(cfg.TgToken, func(msg tg.UserMsg) {
		msgChan <- msg
	})
	if err != nil {
		return nil, fmt.Errorf("tg.GetBot: %w", err)
	}

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
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgChan:
				cc("handleUserMessage", msg, func(ctx context.Context) error {
					return c.handleUserMessage(ctx, msg)
				})
			}
		}
	}()

	apiHandler := api.NewHandler(budgetDomain, cfg.APIAuthToken)

	rootURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("url.Parse(cfg.URL): %w", err)
	}

	tgUpdatesPath := fmt.Sprintf("/%s/%s/%s", rootURL.Path, "tgupdate", cfg.TgToken)

	webHookUrl := fmt.Sprintf("%s://%s%s", rootURL.Scheme, rootURL.Host, tgUpdatesPath)
	log.Println("set TG webhook", webHookUrl)
	wh, err := tgbotapi.NewWebhook(webHookUrl)
	if err != nil {
		log.Fatal("tgbotapi.NewWebhook failed", err.Error())
	}
	_, err = tgBot.API.Request(wh)
	if err != nil {
		log.Fatal("tgAPI.Request(wh) failed", err.Error())
	}
	info, err := tgBot.API.GetWebhookInfo()
	if err != nil {
		log.Fatal("tgAPI.GetWebhookInfo failed", err.Error())
	}
	if info.LastErrorDate != 0 {
		log.Println("info.LastErrorMessage: ", info.LastErrorMessage)
	}

	api.PathPrefix = rootURL.Path + api.PathPrefix
	log.Println("api path prefix: ", api.PathPrefix)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, api.PathPrefix):
			apiHandler.ServeHTTP(writer, request)
			return
		case request.URL.Path == tgUpdatesPath:
			tgBot.HandleUpdateRequest(writer, request)
			return
		default:
			log.Println("invalid URL", request.URL.String())
			writer.WriteHeader(http.StatusNotFound)
			return
		}
	}), nil
}
