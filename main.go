package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	cfg, err := GetConfig()
	fatalIfErr(err)

	account, err := GetAccount(cfg.GmailLogin, cfg.GmailPass)
	fatalIfErr(err)
	defer account.Logout()

	tg, err := GetTelegram(cfg.TgToken, cfg.TgChatID)
	fatalIfErr(err)

	budgets, err := GetBudgets(cfg.GSheetID)
	fatalIfErr(err)

	controller := &Controller{
		budgets:           budgets,
		account:           account,
		telegram:          tg,
		pendingOperations: make(map[string]Operation),
	}

	go http.ListenAndServe("0.0.0.0:8080", nil)
	controller.Run()
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
