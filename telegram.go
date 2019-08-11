package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Telegram is an telegram bot interface
type Telegram interface {
	AskForOperationCategory(operation Operation, btns []Btn) (int, error)
	AskForCustOperationCategory(messageID int, btns []Btn) (int, error)
	SendOperation(operation Operation) error
	GetMessagesChan() <-chan TextMsg
	GetBtnReplyChan() <-chan BtnReply
	SendMessage(text string) error
	AcceptReply(msgID int, text string) error
	AcceptReplyWithBtns(msgID int, btns []Btn) error
}

// TextMsg is a plain text message
type TextMsg struct {
	ID   int
	Text string
}

// Btn is a telegram inline btn
type Btn struct {
	Data string
	Text string
}

// BtnReply is a telegram inline btn reply
type BtnReply struct {
	MessageID int
	Data      string
}

// GetTelegram creates a telegram bot instance
func GetTelegram(botToken string, chatID int64) (Telegram, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	updCfg := tgbotapi.NewUpdate(0)
	updCfg.Timeout = 60
	updCh, err := bot.GetUpdatesChan(updCfg)
	if err != nil {
		return nil, err
	}

	msgChan := make(chan TextMsg)
	btnReplyChan := make(chan BtnReply)

	go func() {
		for upd := range updCh {
			if upd.Message != nil && upd.Message.Chat.ID == chatID {
				msgChan <- TextMsg{ID: upd.Message.MessageID, Text: upd.Message.Text}
			}

			if upd.CallbackQuery != nil {
				btnReplyChan <- BtnReply{
					MessageID: upd.CallbackQuery.Message.MessageID,
					Data:      upd.CallbackQuery.Data,
				}
			}
		}
	}()

	return &telegramImpl{
		bot:          bot,
		chatID:       chatID,
		msgChan:      msgChan,
		btnReplyChan: btnReplyChan,
	}, nil
}

type telegramImpl struct {
	bot          *tgbotapi.BotAPI
	msgChan      chan TextMsg
	btnReplyChan chan BtnReply
	chatID       int64
}

func createStatusInlineKeyboardMarkup(status string) tgbotapi.InlineKeyboardMarkup {
	row := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(status, status),
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	return keyboard
}

func (tg *telegramImpl) AskForOperationCategory(operation Operation, btns []Btn) (int, error) {
	log.Println(operation.Description)

	msgText := fmt.Sprintf("```\n%s\n```\nParsed amount: `%f`", operation.Description, operation.Amount)

	msg := tgbotapi.NewMessage(tg.chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	msg.ReplyMarkup = makeInlineKeyboardMarkup(btns)

	sentMsg, err := tg.bot.Send(msg)

	if err != nil {
		return 0, err
	}

	return sentMsg.MessageID, nil
}

func (tg *telegramImpl) AskForCustOperationCategory(messageID int, btns []Btn) (int, error) {
	msg := tgbotapi.NewMessage(tg.chatID, "Select a category")
	msg.ReplyToMessageID = messageID

	msg.ReplyMarkup = makeInlineKeyboardMarkup(btns)

	sentMsg, err := tg.bot.Send(msg)

	if err != nil {
		return 0, err
	}

	return sentMsg.MessageID, nil
}

func makeInlineKeyboardMarkup(btns []Btn) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, btn := range btns {
		tgBtn := tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data)
		row := []tgbotapi.InlineKeyboardButton{tgBtn}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (tg *telegramImpl) SendOperation(operation Operation) error {
	msgText := fmt.Sprintf("```\n%s\n```\nParsed amount: `%f`", operation.Description, operation.Amount)
	msg := tgbotapi.NewMessage(tg.chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := tg.bot.Send(msg); err != nil {
		return err
	}

	return nil
}

func (tg *telegramImpl) GetMessagesChan() <-chan TextMsg {
	return tg.msgChan
}

func (tg *telegramImpl) GetBtnReplyChan() <-chan BtnReply {
	return tg.btnReplyChan
}

func (tg *telegramImpl) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(tg.chatID, text)
	_, err := tg.bot.Send(msg)

	return err
}

func (tg *telegramImpl) AcceptReply(msgID int, text string) error {
	keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(tg.chatID, msgID, createStatusInlineKeyboardMarkup(text))
	_, err := tg.bot.Send(keyboardEdit)
	return err
}

func (tg *telegramImpl) AcceptReplyWithBtns(msgID int, btns []Btn) error {
	keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(tg.chatID, msgID, makeInlineKeyboardMarkup(btns))
	_, err := tg.bot.Send(keyboardEdit)
	return err
}
