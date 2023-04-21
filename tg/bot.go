package tg

import (
	"encoding/json"
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// GetBot creates a telegram API instance
func GetBot(botToken string, h func(UserMsg)) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		API: bot,
		h:   h,
	}, nil
}

type Bot struct {
	API *tgbotapi.BotAPI
	h   func(UserMsg)
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

	sentMsg, err := b.API.Send(msg)

	if err != nil {
		return 0, fmt.Errorf("API.Send: %w", err)
	}

	return sentMsg.MessageID, nil
}

func (b *Bot) EditBtns(chatID int64, msgID int, newBtns []Btn) error {
	keyboardEdit := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, makeInlineKeyboardMarkup(newBtns))
	_, err := b.API.Send(keyboardEdit)
	if err != nil {
		return fmt.Errorf("API.Send: %w", err)
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

func (b *Bot) HandleUpdateRequest(w http.ResponseWriter, r *http.Request) {
	// Parse incoming request
	upd, err := parseTelegramRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b.h(UserMsg{
		ChatID: upd.Message.Chat.ID,
		ID:     upd.Message.MessageID,
		Text:   upd.Message.Text,
	})
	w.WriteHeader(http.StatusOK)
}

// parseTelegramRequest handles incoming update from the Telegram web hook
func parseTelegramRequest(r *http.Request) (tgbotapi.Update, error) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		return tgbotapi.Update{}, err
	}
	return update, nil
}
