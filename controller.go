package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Controller is an application controller
type Controller struct {
	budgets          Budgets
	account          Account
	telegram         Telegram
	askingOperations map[int]Operation
	googleAuthCfg    *oauth2.Config
	budgetsCache     map[string]string
}

const ignoreBtnID = "ignoreBtnID"

// Run runs the controller
func (c *Controller) Run() {

	msgChan := c.telegram.GetMessagesChan()
	btnReplyChan := c.telegram.GetBtnReplyChan()
	opChan := c.account.GetOperationsChan()

	googleClient := c.getGoogleClient()

	if err := c.budgets.SetClient(googleClient); err != nil {
		log.Fatal(err)
	}

	if err := c.account.SetClient(googleClient); err != nil {
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
		Data: ignoreBtnID,
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
	if reply.Data == ignoreBtnID {
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
func (c *Controller) getGoogleClient() *http.Client {
	tokFile := "token.json"
	tok, err := c.tokenFromFile(tokFile)
	if err != nil {
		tok = c.getTokenFromWeb()
		c.saveToken(tokFile, tok)
	}
	return c.googleAuthCfg.Client(context.Background(), tok)
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
func (c *Controller) getTokenFromWeb() *oauth2.Token {
	authURL := c.googleAuthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	msg := fmt.Sprintf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	c.telegram.SendMessage(msg)

	authCode := <-c.telegram.GetMessagesChan()

	tok, err := c.googleAuthCfg.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}
