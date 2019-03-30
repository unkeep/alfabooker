package main

import (
	"log"
	"strconv"
	"time"
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
