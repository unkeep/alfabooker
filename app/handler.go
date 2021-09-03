package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/unkeep/alfabooker/account"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
)

type handler struct {
	repo    *db.Repo
	tgBot   *tg.Bot
	account *account.Account
	cfg     config
}

func (h *handler) handleNewOperation(ctx context.Context, op account.Operation) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	budget, err := h.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}

	budget.Balance = op.Balance

	h.repo.Budget.Save(ctx, budget)
	if err != nil {
		return fmt.Errorf("Budget.Save: %w", err)
	}

	return nil
}

func (h *handler) handleUserMessage(ctx context.Context, msg tg.UserMsg) error {
	log.Println(msg)

	if msg.ChatID != h.cfg.TgChatID && msg.ChatID != h.cfg.TgAdminChatID {
		return fmt.Errorf("message from unknown chat: %+v", msg)
	}

	text := strings.TrimSpace(msg.Text)

	if text == "?" {
		if err := h.showBudgetStat(ctx, msg.ChatID); err != nil {
			return fmt.Errorf("showBudgetStat: %w", err)
		}
		return nil
	}

	if val, _ := strconv.Atoi(text); val != 0 {
		if err := h.updateBudgetTiming(ctx, val); err != nil {
			return fmt.Errorf("updateBudgetTiming: %w", err)
		}
		return nil
	}

	return nil
}

func (h *handler) updateBudgetTiming(ctx context.Context, days int) error {
	b, err := h.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	now := time.Now()
	b.Amount = b.Balance
	b.StartedAt = now.Unix()
	b.ExpiresAt = now.Add(time.Hour * time.Duration(24*days)).Unix()

	return h.repo.Budget.Save(ctx, b)
}

func (h *handler) showBudgetStat(ctx context.Context, chatID int64) error {
	b, err := h.repo.Budget.Get(ctx)
	if err != nil {
		return fmt.Errorf("Budget.Get: %w", err)
	}

	now := time.Now()
	budgetDuration := b.ExpiresAt - b.StartedAt
	elapsed := now.Unix() - b.StartedAt
	daysToExpiration := time.Unix(b.ExpiresAt, 0).Sub(now).Truncate(time.Hour).Hours() / 24
	estimatedBalance := b.Amount * float64(elapsed/budgetDuration)
	estimatedBalanceDiff := b.Balance - estimatedBalance

	sign := ""
	if estimatedBalanceDiff > 0 {
		sign = "+"
	}
	text := fmt.Sprintf("%dr for %d days (%s%dr estimated)", int(b.Balance), int(daysToExpiration), sign, int(estimatedBalanceDiff))
	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   text,
	}

	if _, err := h.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}
