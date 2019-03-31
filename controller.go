package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Controller struct {
	budgets           Budgets
	account           Account
	telegram          Telegram
	pendingOperations map[string]Operation
}

const IgnoreOptionID = "IgnoreOptionID"

func (c *Controller) Run() {
	opChan := c.pollOperations()
	msgChan := c.telegram.GetMessagesChan()
	opReplyChan := c.telegram.GetOperationReplyChan()

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := c.getClient(config)

	if err := c.budgets.SetClient(client); err != nil {
		log.Fatal(err)
	}

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

	budgets, err := c.budgets.List()
	if err != nil {
		// TODO: handle error
	}

	replyOptions := make([]Option, 0, len(budgets)+1)
	for _, b := range budgets {
		replyOptions = append(replyOptions, Option{
			Data: b.ID,
			Text: b.Name,
		})
	}

	replyOptions = append(replyOptions, Option{
		Data: IgnoreOptionID,
		Text: "❌ Ignore",
	})

	c.telegram.AskForOperationCategory(operation, replyOptions)
}

func (c *Controller) handleNewMessage(msg string) {
}

func (c *Controller) handleOperationReply(opReply OperationReply) {
	op, ok := c.pendingOperations[opReply.OperationID]
	if !ok {
		return
	}

	if opReply.Reply != IgnoreOptionID {
		categoryID, _ := strconv.Atoi(opReply.Reply)
		if err := c.budgets.IncreaseSpent(categoryID, op.Amount); err != nil {
			log.Println(err)
			// TODO: report error
		}
	}

	delete(c.pendingOperations, op.ID)
}

// Retrieve a token, saves the token, then returns the generated client.
func (c *Controller) getClient(config *oauth2.Config) *http.Client {
	tok := c.getTokenFromWeb(config)
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func (c *Controller) getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	msg := fmt.Sprintf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	c.telegram.SendMessage(msg)

	authCode := <-c.telegram.GetMessagesChan()

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}
