package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func main() {
	cfg, err := GetConfig()
	fatalIfErr(err)

	account, err := GetAccount()
	fatalIfErr(err)

	tg, err := GetTelegram(cfg.TgToken, cfg.TgChatID)
	fatalIfErr(err)

	budgets, err := GetBudgets(cfg.GSheetID)
	fatalIfErr(err)

	googleAuthCfg, err := getGoogleAuthConfig(cfg)
	fatalIfErr(err)

	controller := &Controller{
		budgets:          budgets,
		account:          account,
		telegram:         tg,
		askingOperations: make(map[int]Operation),
		budgetsCache:     make(map[string]string),
		googleAuthCfg:    googleAuthCfg,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go http.ListenAndServe("0.0.0.0:"+port, nil)
	controller.Run()
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getGoogleAuthConfig(cfg Config) (*oauth2.Config, error) {
	confJSON := `{"installed":{"client_id":"$client_id","project_id":"$project_id","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"$client_secret","redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]}}`
	confJSON = strings.Replace(confJSON, "$client_id", cfg.GClientID, 1)
	confJSON = strings.Replace(confJSON, "$client_secret", cfg.GClientSecret, 1)
	confJSON = strings.Replace(confJSON, "$project_id", cfg.GProjectID, 1)

	return google.ConfigFromJSON([]byte(confJSON), "https://www.googleapis.com/auth/spreadsheets", gmail.GmailReadonlyScope)
}
