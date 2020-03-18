package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/unkeep/alfabooker/mongo"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Controller is an application controller
type Controller struct {
	budgets       Budgets
	account       Account
	telegram      Telegram
	googleAuthCfg *oauth2.Config
	budgetsCache  map[string]string
	operationsDB  *mongo.OperationCollection
	btnsMetaDB    *mongo.BtnMetaCollection
}

const ignoreBtnCategory = "ignoreCategoryID"

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := c.operationsDB.GetOne(ctx, operation.ID); err != mongo.ErrNotFound {
		return
	}

	if operation.Type == DecreasingOperation && operation.Success {
		err := c.operationsDB.Save(ctx, mongo.Operation{
			ID:      operation.ID,
			Amount:  operation.Amount,
			Balance: operation.Balance,
			RawText: operation.Description,
			Success: operation.Success,
			Time:    time.Now(),
		})
		if err != nil {
			log.Println(err)
		}
		c.askForOperationCategory(operation)
	} else {
		c.telegram.SendOperation(operation)
	}
}

func (c *Controller) butgetsToBtns(opID string, budgets []Budget) []Btn {
	btnMetas := make([]mongo.BtnMeta, 0, len(budgets))
	btns := make([]Btn, 0, len(budgets)+1)

	for _, b := range budgets {
		if strings.HasPrefix(b.Name, ".") {
			continue
		}

		c.budgetsCache[b.ID] = b.Name
		meta := mongo.BtnMeta{
			ActionType:  setCategoryAction,
			OperationID: opID,
			CategotyID:  b.ID,
		}

		btnMetas = append(btnMetas, meta)

		spentPct := int(float32(b.Spent) / float32(b.Amount) * 100.0)
		btns = append(btns, Btn{
			Text: fmt.Sprintf("%s (%d%%)", b.Name, spentPct),
		})
	}

	btnMetas = append(btnMetas, mongo.BtnMeta{
		ActionType:  setCategoryAction,
		OperationID: opID,
		CategotyID:  ignoreBtnCategory,
	})

	btns = append(btns, Btn{
		Text: "❌ Ignore",
	})

	ids, err := c.btnsMetaDB.AddBatch(context.Background(), btnMetas)
	if err != nil {
		log.Println(err)
	}

	for i, id := range ids {
		btns[i].Data = id
	}

	return btns
}

func (c *Controller) askForOperationCategory(operation Operation) {
	budgets, err := c.budgets.List()
	if err != nil {
		log.Println(err)
	}

	btns := c.butgetsToBtns(operation.ID, budgets)

	if _, err := c.telegram.AskForOperationCategory(operation, btns); err != nil {
		log.Println(err)
	}
}

func (c *Controller) handleNewMessage(msg TextMsg) {
	msg.Text = strings.TrimSpace(msg.Text)

	if msg.Text == "?" {
		c.showBudgetsStat()
		return
	}

	if val, _ := strconv.Atoi(msg.Text); val != 0 {
		budgets, err := c.budgets.List()
		if err != nil {
			log.Println(err)
		}

		opID, err := uuid.NewUUID()
		if err != nil {
			log.Println(err)
			return
		}

		op := mongo.Operation{
			ID:      opID.String(),
			Amount:  float64(val),
			RawText: "custom operation: " + msg.Text,
			Success: true,
			Time:    time.Now(),
		}

		if err := c.operationsDB.Save(context.Background(), op); err != nil {
			log.Println("operationsDB.Save: ", err.Error())
			return
		}

		btns := c.butgetsToBtns(op.ID, budgets)

		if _, err := c.telegram.AskForCustOperationCategory(msg.ID, btns); err != nil {
			log.Println("telegram.AskForCustOperationCategory: ", err.Error())
		}
	}
}

func (c *Controller) showBudgetsStat() {
	budgets, err := c.budgets.List()
	if err != nil {
		log.Println(err.Error())
		c.telegram.SendMessage(err.Error())
		return
	}
	lines := make([]string, 0, len(budgets))
	var totalSpent int
	var totalAmount int
	for _, b := range budgets {
		spentPct := int(float32(b.Spent) / float32(b.Amount) * 100)
		name := strings.TrimPrefix(b.Name, ".")
		line := fmt.Sprintf("%s %d/%d(%d%%)", name, b.Spent, b.Amount, spentPct)
		if b.Spent > b.Amount {
			line += "⚠️"
		}
		lines = append(lines, line)

		totalSpent += b.Spent
		totalAmount += b.Amount
	}
	if totalAmount != 0 {
		totalSpentPct := int(float32(totalSpent) / float32(totalAmount) * 100.0)
		line := fmt.Sprintf("TOTAL %d/%d(%d%%)", totalSpent, totalAmount, totalSpentPct)
		lines = append(lines, line)
		lines = append(lines, fmt.Sprintf("BALANCE %d", totalAmount-totalSpent))
	}

	if err := c.telegram.SendMessage(strings.Join(lines, "\n")); err != nil {
		log.Println(err)
	}
}

func (c *Controller) handleBtnReply(reply BtnReply) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	btnMeta, err := c.btnsMetaDB.GetOne(ctx, reply.Data)
	if err != nil {
		log.Println(err)
		return
	}

	op, err := c.operationsDB.GetOne(ctx, btnMeta.OperationID)
	if err != nil {
		log.Println(err)
		return
	}

	var acceptBtns []Btn

	if btnMeta.ActionType == setCategoryAction {
		var acceptingText string
		if btnMeta.CategotyID == ignoreBtnCategory {
			acceptingText = "❌ Ignored"
		} else {
			if err := c.budgets.IncreaseSpent(btnMeta.CategotyID, int(-op.Amount)); err != nil {
				log.Println(err)
				return
			}
			acceptingText = "✅ " + c.budgetsCache[btnMeta.CategotyID]
			op.Category = c.budgetsCache[btnMeta.CategotyID]
			if err := c.operationsDB.Save(ctx, op); err != nil {
				log.Println(err)
			}
		}

		acceptBtnMeta := mongo.BtnMeta{
			ActionType:  editCategoryAction,
			CategotyID:  btnMeta.CategotyID,
			OperationID: op.ID,
		}

		ids, err := c.btnsMetaDB.AddBatch(ctx, []mongo.BtnMeta{acceptBtnMeta})
		if err != nil {
			log.Println(err)
			return
		}

		acceptBtn := Btn{
			Text: acceptingText,
			Data: ids[0],
		}

		acceptBtns = []Btn{acceptBtn}
	} else if btnMeta.ActionType == editCategoryAction {
		if err := c.budgets.IncreaseSpent(btnMeta.CategotyID, -int(op.Amount)); err != nil {
			log.Println(err)
			return
		}

		budgets, err := c.budgets.List()
		if err != nil {
			log.Println(err)
			return
		}

		acceptBtns = c.butgetsToBtns(op.ID, budgets)
	}

	if err := c.telegram.AcceptReplyWithBtns(reply.MessageID, acceptBtns); err != nil {
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

	var authCode string
	for msg := range c.telegram.GetMessagesChan() {
		if len(msg.Text) > 10 {
			authCode = msg.Text
			break
		}
	}

	tok, err := c.googleAuthCfg.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}

const (
	setCategoryAction  = "set"
	editCategoryAction = "edit"
)
