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

	if msg.ChatID != c.cfg.TgAdminChatID {
		return fmt.Errorf("message from unknown chat: %+v", msg)
	}

	text := strings.TrimSpace(msg.Text)
	text = strings.ToLower(text)

	if text == "/help" {
		if err := c.showHelp(ctx, msg.ChatID); err != nil {
			return fmt.Errorf("showHelp: %w", err)
		}
	}

	if text == "?" {
		if err := c.showBudgetStat(ctx, msg.ChatID); err != nil {
			return fmt.Errorf("showBudgetStat: %w", err)
		}
		return nil
	}

	if strings.HasPrefix(text, "start ") {
		text = strings.TrimPrefix(text, "start ")
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

	if strings.HasPrefix(text, "card ") {
		text = strings.TrimPrefix(text, "card ")
		val, err := strconv.Atoi(text)
		if err != nil {
			return fmt.Errorf("parse account value: %w", err)
		}

		if err := c.budgetDomain.UpdateAccountBalance(ctx, float64(val)); err != nil {
			return fmt.Errorf("budgetDomain.UpdateAccountBalance: %w", err)
		}

		return nil
	}

	if strings.HasPrefix(text, "align ") {
		text = strings.TrimPrefix(text, "align ")
		val, err := strconv.Atoi(text)
		if err != nil {
			return fmt.Errorf("parse align value: %w", err)
		}

		if err := c.budgetDomain.DecreaseAndAlignBudget(ctx, float64(val)); err != nil {
			return fmt.Errorf("budgetDomain.DecreaseAndAlignBudget: %w", err)
		}

		return nil
	}

	if strings.HasPrefix(text, "reserve ") {
		text = strings.TrimPrefix(text, "reserve ")
		val, err := strconv.Atoi(text)
		if err != nil {
			return fmt.Errorf("parse reselved value: %w", err)
		}

		if err := c.budgetDomain.SetReservedValue(ctx, float64(val)); err != nil {
			return fmt.Errorf("budgetDomain.SetReservedValue: %w", err)
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

func (c *controller) showHelp(ctx context.Context, chatID int64) error {
	msgText := `
?           - show statistics

start <num>   - start new budget tracking for <num> days 

card          - set amount on card to <num> 

cash <num>    - set amount of cash to <num> 

<num>         - decrease cache by <num>

align <num>   - decrease budget amount by <num> and it's duration proportionately'

reserve <num> - set reserved balance value (will not be counted in total balance)
`

	msgText = strings.TrimPrefix(msgText, "\n")

	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   msgText,
	}
	if _, err := c.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil

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

	text := fmt.Sprintf(`
card: %d, cash: %d, reserved: %d
total: %d
%s from estimated balance
%.1f days left
%d avg daily spending`,
		int(stat.AccountBalance),
		int(stat.CashBalance),
		int(stat.ReservedBalance),
		int(stat.TotalBalance),
		balanceDeviationStr,
		stat.BudgetDaysToExpiration,
		int(stat.DailyAverageSpending),
	)
	text = strings.TrimPrefix(text, "\n")

	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   text,
	}

	if _, err := c.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}
