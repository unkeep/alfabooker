package main

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var IgnoreOption = "/ignore"

type Telegram interface {
	AskForOperationCategory(operation Operation, options []string) error
	GetMessagesChan() <-chan string
	GetOperationReplyChan() <-chan OperationReply
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
				bot.AnswerCallbackQuery(tgbotapi.NewCallback(upd.CallbackQuery.ID, upd.CallbackQuery.Data))

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

func createOperationReplyButton(operation Operation, option string) tgbotapi.InlineKeyboardButton {
	data := operation.ID + ":" + option
	return tgbotapi.NewInlineKeyboardButtonData(option, data)
}

func (tg *telegramImpl) AskForOperationCategory(operation Operation, options []string) error {
	log.Println(operation.Description)

	msgText := fmt.Sprintf("```\n%s\n```\nParsed amount: `%f`", operation.Description, operation.Amount)

	msg := tgbotapi.NewMessage(tg.chatID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	var buttons []tgbotapi.InlineKeyboardButton
	for _, option := range options {
		buttons = append(buttons, createOperationReplyButton(operation, option))
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)

	_, err := tg.bot.Send(msg)

	return err
}

func (tg *telegramImpl) GetMessagesChan() <-chan string {
	return tg.msgChan
}

func (tg *telegramImpl) GetOperationReplyChan() <-chan OperationReply {
	return tg.opReplyChan
}
