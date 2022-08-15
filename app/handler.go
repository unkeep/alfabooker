package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/unkeep/alfabooker/budget"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
)

type controller struct {
	repo  *db.Repo
	tgBot *tg.Bot
	cfg   config

	budgetDomain *budget.Domain
}

func (c *controller) handleUserMessage(ctx context.Context, msg tg.UserMsg) error {
	log.Println(msg)

	if msg.ChatID != c.cfg.TgChatID && msg.ChatID != c.cfg.TgAdminChatID {
		return fmt.Errorf("message from unknown chat: %+v", msg)
	}

	text := strings.TrimSpace(msg.Text)

	if text == "?" {
		if err := c.showBudgetStat(ctx, msg.ChatID); err != nil {
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

		if err := c.updateBudgetTiming(ctx, val); err != nil {
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

		if err := c.setCash(ctx, val); err != nil {
			return fmt.Errorf("setCash: %w", err)
		}
		return nil
	}

	if val, err := strconv.Atoi(text); err != nil {
		if err := c.decreaseCash(ctx, val); err != nil {
			return fmt.Errorf("decreaseCash: %w", err)
		}
		return nil
	}

	return nil
}

func (c *controller) decreaseCash(ctx context.Context, val int) error {
	b, err := c.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	b.CashBalance -= float64(val)

	return c.repo.Budget.Save(ctx, b)
}

func (c *controller) setCash(ctx context.Context, val int) error {
	b, err := c.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	b.CashBalance = float64(val)

	return c.repo.Budget.Save(ctx, b)
}

func (c *controller) updateBudgetTiming(ctx context.Context, days int) error {
	b, err := c.repo.Budget.Get(ctx)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("Budget.Get: %w", err)
	}
	now := time.Now()
	b.Amount = b.Balance + b.CashBalance
	b.StartedAt = now.Unix()
	b.ExpiresAt = now.Add(time.Hour * time.Duration(24*days)).Unix()

	return c.repo.Budget.Save(ctx, b)
}

func (c *controller) showBudgetStat(ctx context.Context, chatID int64) error {
	stat, err := c.budgetDomain.GetStat(ctx)
	if err != nil {
		return fmt.Errorf("budgetDomain.GetStat: %w", err)
	}

	balanceDeviationStr := fmt.Sprint(int(stat.BalanceDeviation))
	if int(stat.BalanceDeviation) > 0 {
		balanceDeviationStr = "+" + balanceDeviationStr
	}

	text := fmt.Sprintf("осталось %d (карта: %d, кеш: %d) на %.1f дней (%s от запланированного)",
		int(stat.TotalBalance),
		int(stat.AccountBalance),
		int(stat.CashBalance),
		stat.BudgetDaysToExpiration,
		balanceDeviationStr,
	)
	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   text,
	}

	if _, err := c.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}
