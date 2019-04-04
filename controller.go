package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Controller struct {
	budgets          Budgets
	account          Account
	telegram         Telegram
	askingOperations map[int]Operation
	googleAuthCode   string
	budgetsCache     map[string]string
}

const IgnoreBtnID = "IgnoreBtnID"

func (c *Controller) Run() {
	opChan := c.pollOperations()
	msgChan := c.telegram.GetMessagesChan()
	btnReplyChan := c.telegram.GetBtnReplyChan()

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
		case btnReply := <-btnReplyChan:
			c.handleBtnReply(btnReply)
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
	budgets, err := c.budgets.List()
	if err != nil {
		// TODO: handle error
	}

	btns := make([]Btn, 0, len(budgets)+1)
	for _, b := range budgets {
		c.budgetsCache[b.ID] = b.Name
		btns = append(btns, Btn{
			Data: b.ID,
			Text: fmt.Sprintf("%s (%d%%)", b.Name, b.SpentPct),
		})
	}

	btns = append(btns, Btn{
		Data: IgnoreBtnID,
		Text: "❌ Ignore",
	})

	if msgID, err := c.telegram.AskForOperationCategory(operation, btns); err == nil {
		c.askingOperations[msgID] = operation
	} else {
		log.Println(err)
	}
}

func (c *Controller) handleNewMessage(msg string) {
}

func (c *Controller) handleBtnReply(reply BtnReply) {
	op, ok := c.askingOperations[reply.MessageID]
	if !ok {
		return
	}

	var acceptingText string
	if reply.Data == IgnoreBtnID {
		acceptingText = "❌ Ignored"
	} else {
		budgetID := reply.Data
		if err := c.budgets.IncreaseSpent(budgetID, op.Amount); err != nil {
			log.Println(err)
			return
		}
		acceptingText = "✅ " + c.budgetsCache[budgetID]
	}

	delete(c.askingOperations, reply.MessageID)

	if err := c.telegram.AcceptReply(reply.MessageID, acceptingText); err != nil {
		log.Println(err)
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func (c *Controller) getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := c.tokenFromFile(tokFile)
	if err != nil {
		tok = c.getTokenFromWeb(config)
		c.saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func (c *Controller) tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func (c *Controller) saveToken(path string, token *oauth2.Token) {
	log.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
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
