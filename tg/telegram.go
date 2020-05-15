package tg

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// GetGroup creates a telegram bot instance
func GetGroup(botToken string, chatID int64) (Group, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	return &telegramImpl{
		bot:    bot,
		chatID: chatID,
	}, nil
}

type Group struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

func (g *Group) SendMessage(m BotMessage) (int, error) {
	msg := tgbotapi.NewMessage(g.chatID, m.Text)
	if m.TextMarkdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}
	msg.ReplyToMessageID = m.ReplyToMsgID
	if m.Btns != nil {
		msg.ReplyMarkup = makeInlineKeyboardMarkup(m.Btns)
	}

	sentMsg, err := g.bot.Send(msg)

	if err != nil {
		return 0, fmt.Errorf(err, "bot.Send: %w", err)
	}

	return sentMsg.MessageID, nil
}

func (g *Group) EditBtns(msgID int, newBtns []Btn) error {
	keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(g.chatID, msgID, makeInlineKeyboardMarkup(newBtns))
	_, err := g.bot.Send(keyboardEdit)
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

func (g *Group) GetUpdates(ctx context.Context, msgs chan<- UserMsg, clicks chan<- BtnClick) error {
	updCfg := tgbotapi.NewUpdate(0)
	updCfg.Timeout = 60
	updCh, err := g.bot.GetUpdatesChan(updCfg)
	if err != nil {
		return nil, fmt.Errorf("bot.GetUpdatesChan: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case upd := <-updCh:
			if upd.Message != nil && upd.Message.Chat.ID == chatID {
				msgs <- UserMsg{ID: upd.Message.MessageID, Text: upd.Message.Text}
			}

			if upd.CallbackQuery != nil {
				clicks <- BtnClick{
					MessageID: upd.CallbackQuery.Message.MessageID,
					BtnID:     upd.CallbackQuery.Data,
				}
			}
		}
	}
}
