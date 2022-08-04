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

	if strings.HasPrefix(text, "new ") {
		text = strings.TrimPrefix(text, "new ")
		val, err := strconv.Atoi(text)
		if err != nil {
			return fmt.Errorf("parse days: %w", err)
		}

		if err := h.updateBudgetTiming(ctx, val); err != nil {
			return fmt.Errorf("updateBudgetTiming: %w", err)
		}
		return nil
	}

	if strings.HasPrefix(text, "cash ") {
		text = strings.TrimPrefix(text, "cash ")
		val, err := strconv.Atoi(text)
		if err != nil {
			return fmt.Errorf("parse cash value: %w", err)
		}

		if err := h.setCash(ctx, val); err != nil {
			return fmt.Errorf("setCash: %w", err)
		}
		return nil
	}

	if val, err := strconv.Atoi(text); err != nil {
		if err := h.decreaseCash(ctx, val); err != nil {
			return fmt.Errorf("decreaseCash: %w", err)
		}
		return nil
	}

	return nil
}

func (h *handler) decreaseCash(ctx context.Context, val int) error {
	b, err := h.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	b.CashBalance -= float64(val)

	return h.repo.Budget.Save(ctx, b)
}

func (h *handler) setCash(ctx context.Context, val int) error {
	b, err := h.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	b.CashBalance = float64(val)

	return h.repo.Budget.Save(ctx, b)
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
	budgetDuration := float64(b.ExpiresAt - b.StartedAt)
	elapsed := float64(now.Unix() - b.StartedAt)
	daysToExpiration := time.Unix(b.ExpiresAt, 0).Sub(now).Hours() / 24.0
	estimatedSpending := b.Amount * elapsed / budgetDuration
	actualSpending := b.Amount - b.Balance - b.CashBalance
	spendingDiff := estimatedSpending - actualSpending

	sign := ""
	if spendingDiff > 0 {
		sign = "+"
	}
	text := fmt.Sprintf("осталось %dGEL(карта: %d, кеш: %d) на %.1f дней (%s%dGEL от запланированного)",
		int(b.Balance+b.CashBalance), int(b.Balance), int(b.CashBalance), daysToExpiration, sign, int(spendingDiff))
	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   text,
	}

	if _, err := h.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}
