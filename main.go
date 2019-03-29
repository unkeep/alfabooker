package main

import (
	"log"
	"time"
)

func GetBudgets() (Budgets, error) {
	return nil, nil
}

type Controller struct {
	budgets           Budgets
	account           Account
	telegram          Telegram
	pendingOperations map[string]Operation
}

func (c *Controller) Run() {
	opChan := c.pollOperations()
	msgChan := c.telegram.GetMessagesChan()
	opReplyChan := c.telegram.GetOperationReplyChan()

	for {
		select {
		case op := <-opChan:
			c.handleNewOperation(op)
		case msg := <-msgChan:
			c.handleNewMessage(msg)
		case opReply := <-opReplyChan:
			c.handleOperationReply(opReply)
		}
	}
}

func (c *Controller) pollOperations() chan Operation {
	ch := make(chan Operation)

	go func() {
		for {
			op, err := c.account.GetLastOperation()
			if err == nil {
				ch <- op
				continue
			}

			if err != ErrOperationNotFound {
				// TODO: handle err
				log.Println(err)
			}

			time.Sleep(time.Second * 5)
		}
	}()

	return ch
}

func (c *Controller) handleNewOperation(operation Operation) {
	c.pendingOperations[operation.ID] = operation

	// TODO: get categories
	categories := []string{}

	options := append(categories, IgnoreOption)

	c.telegram.AskForOperationCategory(operation, options)
}

func (c *Controller) handleNewMessage(msg string) {
}

func (c *Controller) handleOperationReply(opReply OperationReply) {
	op, ok := c.pendingOperations[opReply.OperationID]
	if !ok {
		// TODO: report error
		return
	}

	log.Printf("Operation: %v, reply: %s\n", op, opReply.Reply)
}

func main() {
	cfg, err := GetConfig()
	fatalIfErr(err)

	account, err := GetAccount(cfg.GmailLogin, cfg.GmailPass)
	fatalIfErr(err)
	defer account.Logout()

	tg, err := GetTelegram(cfg.TgToken, cfg.TgChatID)
	fatalIfErr(err)

	budgets, err := GetBudgets()
	fatalIfErr(err)

	controller := &Controller{
		budgets:           budgets,
		account:           account,
		telegram:          tg,
		pendingOperations: make(map[string]Operation),
	}

	controller.Run()
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
