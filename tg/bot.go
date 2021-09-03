package tg

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// GetBot creates a telegram bot instance
func GetBot(botToken string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot: bot,
	}, nil
}

type Bot struct {
	bot *tgbotapi.BotAPI
}

func (b *Bot) SendMessage(m BotMessage) (int, error) {
	msg := tgbotapi.NewMessage(m.ChatID, m.Text)
	if m.TextMarkdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}
	msg.ReplyToMessageID = m.ReplyToMsgID
	if m.Btns != nil {
		msg.ReplyMarkup = makeInlineKeyboardMarkup(m.Btns)
	}

	sentMsg, err := b.bot.Send(msg)

	if err != nil {
		return 0, fmt.Errorf("bot.Send: %w", err)
	}

	return sentMsg.MessageID, nil
}

func (b *Bot) EditBtns(chatID int64, msgID int, newBtns []Btn) error {
	keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, makeInlineKeyboardMarkup(newBtns))
	_, err := b.bot.Send(keyboardEdit)
	if err != nil {
		return fmt.Errorf("bot.Send: %w", err)
	}

	return nil
}

func makeInlineKeyboardMarkup(btns []Btn) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, btn := range btns {
		tgBtn := tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.ID)
		row := []tgbotapi.InlineKeyboardButton{tgBtn}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) GetUpdates(ctx context.Context, msgs chan<- UserMsg) error {
	updCfg := tgbotapi.NewUpdate(0)
	updCfg.Timeout = 60
	updCh, err := b.bot.GetUpdatesChan(updCfg)
	if err != nil {
		return fmt.Errorf("bot.GetUpdatesChan: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-updCh:
			if upd.Message != nil {
				msgs <- UserMsg{
					ChatID: upd.Message.Chat.ID,
					ID:     upd.Message.MessageID,
					Text:   upd.Message.Text,
				}
			}
		}
	}
}
