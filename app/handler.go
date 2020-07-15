package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/unkeep/alfabooker/account"
	"github.com/unkeep/alfabooker/budget"
	"github.com/unkeep/alfabooker/db"
	"github.com/unkeep/alfabooker/tg"
)

type handler struct {
	repo    *db.Repo
	tgBot   *tg.Bot
	budgets *budget.Budgets
	account *account.Account
	cfg     config
}

var ErrOperationAlreadyHandled = fmt.Errorf("operationAlreadyHandled")

func (h *handler) handleNewOperation(ctx context.Context, op account.Operation) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if _, err := h.repo.Operations.GetOne(ctx, op.ID); err == nil {
		return ErrOperationAlreadyHandled
	} else if err != db.ErrNotFound {
		return fmt.Errorf("Operations.GetOne: %w", err)
	}

	dbOp := db.Operation{
		ID:          op.ID,
		Amount:      op.Amount,
		Balance:     op.Balance,
		RawText:     op.RawText,
		Description: op.Description,
		Success:     op.Success,
		Time:        time.Now(),
	}

	err := h.repo.Operations.Save(ctx, dbOp)
	if err != nil {
		return fmt.Errorf("Operations.Save(%+v): %w", dbOp, err)
	}

	msg := tg.BotMessage{
		ChatID:       h.cfg.TgChatID,
		Text:         fmt.Sprintf("```\n%s\n```\nParsed amount: `%f`", op.RawText, op.Amount),
		TextMarkdown: true,
	}

	if op.Amount < 0 && op.Success {
		btns, err := h.makeOpBtns(op.ID, h.cfg.TgChatID)
		if err != nil {
			return fmt.Errorf("makeOpBtns: %w", err)
		}
		msg.Btns = btns
	}

	if _, err := h.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}

const ignoreBtnCategory = "ignoreCategoryID"

func (h *handler) makeOpBtns(opID string, chatID int64) ([]tg.Btn, error) {
	budgets, err := h.budgets.List()
	if err != nil {
		return nil, fmt.Errorf("budgets.List: %w", err)
	}

	btnMetas := make([]db.BtnMeta, 0, len(budgets))
	btns := make([]tg.Btn, 0, len(budgets)+1)

	for _, b := range budgets {
		if strings.HasPrefix(b.Name, ".") {
			continue
		}

		meta := db.BtnMeta{
			ActionType:   setCategoryAction,
			OperationID:  opID,
			CategotyID:   b.ID,
			CategotyName: b.Name,
			ChatID:       chatID,
		}

		btnMetas = append(btnMetas, meta)

		spentPct := int(float32(b.Spent) / float32(b.Amount) * 100.0)
		btns = append(btns, tg.Btn{
			Text: fmt.Sprintf("%s (%d%%)", b.Name, spentPct),
		})
	}

	btnMetas = append(btnMetas, db.BtnMeta{
		ActionType:   setCategoryAction,
		OperationID:  opID,
		CategotyID:   ignoreBtnCategory,
		CategotyName: "❌ Ignored",
		ChatID:       chatID,
	})

	btns = append(btns, tg.Btn{
		Text: "❌ Ignore",
	})

	ids, err := h.repo.BtnMetas.AddBatch(context.Background(), btnMetas)
	if err != nil {
		return nil, fmt.Errorf("BtnMetas.AddBatch: %w", err)
	}

	for i, id := range ids {
		btns[i].ID = id
	}

	return btns, nil
}

func (h *handler) handleUserMessage(ctx context.Context, msg tg.UserMsg) error {
	log.Println(msg)

	if msg.ChatID != h.cfg.TgChatID && msg.ChatID != h.cfg.TgAdminChatID {
		return fmt.Errorf("message from unknown chat: %+v", msg)
	}

	text := strings.TrimSpace(msg.Text)

	if text == "?" {
		if err := h.showBudgetsStat(msg.ChatID); err != nil {
			return fmt.Errorf("showBudgetsStat: %w", err)
		}
		return nil
	}

	if val, _ := strconv.Atoi(text); val != 0 {
		if err := h.handleCustomOperation(ctx, val, msg); err != nil {
			return fmt.Errorf("handleCustomOperation: %w", err)
		}
		return nil
	}

	return nil
}

func (h *handler) handleCustomOperation(ctx context.Context, ammount int, msg tg.UserMsg) error {
	opID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("uuid.NewUUID(): %w", err)
	}

	op := db.Operation{
		ID:      opID.String(),
		Amount:  -float64(ammount),
		RawText: fmt.Sprintf("custom operation: %d", ammount),
		Success: true,
		Time:    time.Now(),
	}

	if err := h.repo.Operations.Save(ctx, op); err != nil {
		return fmt.Errorf("Operations.Save: %w", err)
	}

	btns, err := h.makeOpBtns(op.ID, msg.ChatID)
	if err != nil {
		return fmt.Errorf("makeOpBtns: %w", err)
	}

	botMsg := tg.BotMessage{
		ChatID:       msg.ChatID,
		Text:         "Select a category",
		ReplyToMsgID: msg.ID,
		Btns:         btns,
	}

	if _, err := h.tgBot.SendMessage(botMsg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}

func (h *handler) showBudgetsStat(chatID int64) error {
	budgets, err := h.budgets.List()
	if err != nil {
		return fmt.Errorf("budgets.List: %w", err)
	}

	lines := make([]string, 0, len(budgets))
	for _, b := range budgets {
		spentPct := int(float32(b.Spent) / float32(b.Amount) * 100)
		name := strings.TrimPrefix(b.Name, ".")
		line := fmt.Sprintf("%s %d/%d(%d%%)", name, b.Spent, b.Amount, spentPct)
		if b.Spent > b.Amount {
			line += "⚠️"
		}
		lines = append(lines, line)
	}

	// totals
	lines = append(lines, "")

	tatals, err := h.budgets.Totals()
	if err != nil {
		return fmt.Errorf("budgets.Totals: %w", err)
	}
	lines = append(lines, tatals...)

	msg := tg.BotMessage{
		ChatID: chatID,
		Text:   strings.Join(lines, "\n"),
	}

	if _, err := h.tgBot.SendMessage(msg); err != nil {
		return fmt.Errorf("tgBot.SendMessage: %w", err)
	}

	return nil
}

const (
	setCategoryAction  = "set"
	editCategoryAction = "edit"
)

func (h *handler) handleBtnClick(ctx context.Context, btnClick tg.BtnClick) error {
	btnMeta, err := h.repo.BtnMetas.GetOne(ctx, btnClick.BtnID)
	if err != nil {
		return fmt.Errorf("BtnMetas.GetOne: %w", err)
	}

	op, err := h.repo.Operations.GetOne(ctx, btnMeta.OperationID)
	if err != nil {
		return fmt.Errorf("Operations.GetOne: %w", err)
	}

	var newBtns []tg.Btn
	if btnMeta.ActionType == setCategoryAction {
		var acceptingText string
		if btnMeta.CategotyID == ignoreBtnCategory {
			acceptingText = "❌ Ignored"
		} else {
			if err := h.budgets.IncreaseSpent(btnMeta.CategotyID, -int(op.Amount)); err != nil {
				return fmt.Errorf("budgets.IncreaseSpent: %w", err)
			}
			acceptingText = "✅ " + btnMeta.CategotyName
			if err := h.repo.Operations.Save(ctx, op); err != nil {
				return fmt.Errorf("Operations.Save %w", err)
			}
		}

		acceptingBtnMeta := btnMeta
		acceptingBtnMeta.ActionType = editCategoryAction

		ids, err := h.repo.BtnMetas.AddBatch(ctx, []db.BtnMeta{acceptingBtnMeta})
		if err != nil {
			return fmt.Errorf("BtnMetas.AddBatch: %w", err)
		}

		acceptBtn := tg.Btn{
			Text: acceptingText,
			ID:   ids[0],
		}
		newBtns = []tg.Btn{acceptBtn}
	} else if btnMeta.ActionType == editCategoryAction {
		if btnMeta.CategotyID != ignoreBtnCategory {
			if err := h.budgets.IncreaseSpent(btnMeta.CategotyID, int(op.Amount)); err != nil {
				return fmt.Errorf("budgets.IncreaseSpent: %w", err)
			}
		}

		var err error
		newBtns, err = h.makeOpBtns(op.ID, btnMeta.ChatID)
		if err != nil {
			return fmt.Errorf("makeOpBtns: %w", err)
		}
	}

	if err := h.tgBot.EditBtns(btnMeta.ChatID, btnClick.MessageID, newBtns); err != nil {
		return fmt.Errorf("tgBot.EditBtns: %w", err)
	}

	return nil
}
