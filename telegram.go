package main

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Option struct {
	Data string
	Text string
}

type Telegram interface {
	AskForOperationCategory(operation Operation, options []Option) error
	GetMessagesChan() <-chan string
	GetOperationReplyChan() <-chan OperationReply
	SendMessage(text string) error
}

type OperationReply struct {
	OperationID string
	Reply       string
}

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

	msgChan := make(chan string)
	opReplyChan := make(chan OperationReply)

	go func() {
		for upd := range updCh {
			if upd.Message != nil && upd.Message.Chat.ID == chatID {
				msgChan <- upd.Message.Text
			}

			if upd.CallbackQuery != nil {
				data := upd.CallbackQuery.Data
				replyTokens := strings.Split(data, ":")
				if len(replyTokens) != 2 {
					// TODO: report error
					continue
				}

				opReplyChan <- OperationReply{
					OperationID: replyTokens[0],
					Reply:       replyTokens[1],
				}

				keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(chatID, upd.CallbackQuery.Message.MessageID, createStatusInlineKeyboardMarkup("✅ saved"))
				_, err := bot.Send(keyboardEdit)
				if err != nil {
					log.Println(err)
				}
				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, "Category accepted"))
			}
		}
	}()

	return &telegramImpl{
		bot:         bot,
		chatID:      chatID,
		msgChan:     msgChan,
		opReplyChan: opReplyChan,
	}, nil
}

type telegramImpl struct {
	bot         *tgbotapi.BotAPI
	msgChan     chan string
	opReplyChan chan OperationReply
	chatID      int64
}

func createStatusInlineKeyboardMarkup(status string) tgbotapi.InlineKeyboardMarkup {
	row := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(status, status),
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	return keyboard
}

func createOperationReplyButton(operation Operation, option Option) tgbotapi.InlineKeyboardButton {
	data := operation.ID + ":" + option.Data
	return tgbotapi.NewInlineKeyboardButtonData(option.Text, data)
}

func (tg *telegramImpl) AskForOperationCategory(operation Operation, options []Option) error {
	log.Println(operation.Description)

	msgText := fmt.Sprintf("```\n%s\n```\nParsed amount: `%f`", operation.Description, operation.Amount)

	msg := tgbotapi.NewMessage(tg.chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, option := range options {
		if len(row) == 2 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0, 2)
		}
		btn := createOperationReplyButton(operation, option)
		row = append(row, btn)
	}

	if len(row) != 0 {
		rows = append(rows, row)
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := tg.bot.Send(msg)

	return err
}

func (tg *telegramImpl) GetMessagesChan() <-chan string {
	return tg.msgChan
}

func (tg *telegramImpl) GetOperationReplyChan() <-chan OperationReply {
	return tg.opReplyChan
}

func (tg *telegramImpl) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(tg.chatID, text)
	_, err := tg.bot.Send(msg)

	return err
}
