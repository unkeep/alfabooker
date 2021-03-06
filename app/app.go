package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/unkeep/alfabooker/account"
	"github.com/unkeep/alfabooker/budget"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
	"golang.org/x/oauth2"
)

const googleTokenID = "google_auth"

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

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpServer := http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: http.HandlerFunc(healthcheckHandler),
	}

	go httpServer.ListenAndServe()
	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	googleAuthCfg, err := getGoogleAuthConfig(cfg)
	if err != nil {
		return fmt.Errorf("getGoogleAuthConfig: %w", err)
	}

	log.Println("GetRepo")
	repo, err := db.GetRepo(ctx, cfg.MongoURI)
	if err != nil {
		return fmt.Errorf("db.GetRepo: %w", err)
	}

	log.Println("GetBot")
	tgBot, err := tg.GetBot(cfg.TgToken)
	if err != nil {
		return fmt.Errorf("tg.GetBot: %w", err)
	}

	msgChan := make(chan tg.UserMsg, 0)
	btnClicksChan := make(chan tg.BtnClick, 0)
	critErrosChan := make(chan error, 0)

	go func() {
		if err := tgBot.GetUpdates(ctx, msgChan, btnClicksChan); err != nil {
			critErrosChan <- fmt.Errorf("tgBot.GetUpdates: %w", err)
		}
	}()

	googleAutToken, err := getGoogleAuthToken(ctx, googleAuthCfg, tgBot, msgChan, cfg.TgAdminChatID, repo.Tokens)
	if err != nil {
		return fmt.Errorf("getGoogleAuthToken: %w", err)
	}

	googleClient := googleAuthCfg.Client(ctx, googleAutToken)

	log.Println("budget.New")
	budgets, err := budget.New(googleClient, cfg.GSheetID)
	if err != nil {
		return fmt.Errorf("budget.New: %w", err)
	}

	log.Println("account.New")
	acc, err := account.New(googleClient)
	if err != nil {
		return fmt.Errorf("account.New: %w", err)
	}

	opChan := make(chan account.Operation, 0)
	go func() {
		log.Println("acc.GetOperations")
		if err := acc.GetOperations(ctx, opChan); err != nil {
			critErrosChan <- fmt.Errorf("acc.GetOperations: %w", err)
		}
	}()

	h := handler{
		cfg:     cfg,
		repo:    repo,
		account: acc,
		budgets: budgets,
		tgBot:   tgBot,
	}

	hh := func(name string, param interface{}, f func(ctx context.Context) error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		if err := f(ctx); err != nil {
			log.Printf("%s(%+v): %s\n", name, param, err.Error())
			tgBot.SendMessage(tg.BotMessage{
				ChatID: cfg.TgAdminChatID,
				Text: fmt.Sprintf("⚠️ handler: %s, error:\n```%s```\ncontext:\n```%+v```\n",
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
		case op := <-opChan:
			hh("handleNewOperation", op, func(ctx context.Context) error {
				return h.handleNewOperation(ctx, op)
			})
		case msg := <-msgChan:
			hh("handleUserMessage", msg, func(ctx context.Context) error {
				return h.handleUserMessage(ctx, msg)
			})
		case btnClick := <-btnClicksChan:
			hh("handleBtnClick", btnClick, func(ctx context.Context) error {
				return h.handleBtnClick(ctx, btnClick)
			})
		}
	}
}

func getGoogleAuthToken(
	ctx context.Context,
	googleAuthCfg *oauth2.Config,
	tgBot *tg.Bot,
	msgChan <-chan tg.UserMsg,
	adminChatID int64,
	tokensRepo *db.TokensRepo) (*oauth2.Token, error) {

	savedGoogleTok, err := tokensRepo.GetOne(ctx, googleTokenID)
	if err != nil {
		if err != db.ErrNotFound {
			log.Println(fmt.Errorf("failed to get google auth token from db: %w", err))
		}
		return getGoogleAuthTokenFromTg(ctx, googleAuthCfg, tgBot, msgChan, adminChatID, tokensRepo)
	}

	var googleTok *oauth2.Token
	if err := json.Unmarshal(savedGoogleTok.Data, &googleTok); err != nil {
		log.Println(fmt.Errorf("failed to unmarshal google auth token from db: %w", err))
		return getGoogleAuthTokenFromTg(ctx, googleAuthCfg, tgBot, msgChan, adminChatID, tokensRepo)
	}

	if !googleTok.Valid() {
		googleTok, err = googleAuthCfg.TokenSource(ctx, googleTok).Token()
		if err != nil {
			log.Println(fmt.Errorf("failed to refresh google token: %w", err))
			return getGoogleAuthTokenFromTg(ctx, googleAuthCfg, tgBot, msgChan, adminChatID, tokensRepo)
		}
	}

	return googleTok, nil
}

func getGoogleAuthTokenFromTg(
	ctx context.Context,
	googleAuthCfg *oauth2.Config,
	tgBot *tg.Bot,
	msgChan <-chan tg.UserMsg,
	adminChatID int64,
	tokensRepo *db.TokensRepo) (*oauth2.Token, error) {

	authURL := googleAuthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	authLinkMsg := tg.BotMessage{
		ChatID: adminChatID,
		Text:   fmt.Sprintf("Go to the following link in your browser then type the authorization code: \n%s\n", authURL),
	}
	if _, err := tgBot.SendMessage(authLinkMsg); err != nil {
		return nil, fmt.Errorf("tgBot.SendMessage(authLinkMsg): %w", err)
	}

	log.Println("waitForAuthCode")
	code := waitForAuthCode(msgChan, adminChatID)
	if code == "" {
		return nil, fmt.Errorf("failed to get google auth code")
	}

	log.Println("googleAuthCfg.Exchange")
	tok, err := googleAuthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("googleAuthCfg.Exchange: %w", err)
	}

	tokData, err := json.Marshal(tok)
	if err != nil {
		return tok, fmt.Errorf("failed to marshal google auth token: %w", err)
	}
	if err := tokensRepo.Save(ctx, db.Token{
		ID:   googleTokenID,
		Data: tokData,
		Time: time.Now(),
	}); err != nil {
		return tok, fmt.Errorf("failed to save google auth token: %w", err)
	}

	return tok, nil
}

func waitForAuthCode(msgChan <-chan tg.UserMsg, adminChatID int64) string {
	for msg := range msgChan {
		if msg.ChatID == adminChatID {
			return msg.Text
		}
	}
	return ""
}
