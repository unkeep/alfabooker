package budget

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeep/alfabooker/db"
)

type Domain struct {
	BudgetRepo *db.BudgetRepo
}

func (d *Domain) GetStat(ctx context.Context) (*Statistics, error) {
	b, err := d.BudgetRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("BudgetRepo.Get: %w", err)
	}

	now := time.Now()
	elapsed := float64(now.Unix() - b.StartedAt)
	budgetDuration := float64(b.ExpiresAt - b.StartedAt)
	daysToExpiration := time.Unix(b.ExpiresAt, 0).Sub(now).Hours() / 24.0

	estimatedSpendingCoeff := b.Amount / budgetDuration
	estimatedSpending := elapsed * estimatedSpendingCoeff
	estimatedBalance := b.Amount - estimatedSpending

	totalBalance := b.Balance + b.CashBalance

	balanceDeviation := totalBalance - estimatedBalance

	return &Statistics{
		BudgetAmount:           b.Amount,
		BudgetStartedAt:        b.StartedAt,
		BudgetExpiresAt:        b.ExpiresAt,
		BudgetDaysToExpiration: daysToExpiration,
		AccountBalance:         b.Balance,
		CashBalance:            b.CashBalance,
		TotalBalance:           totalBalance,
		EstimatedBalance:       estimatedBalance,
		BalanceDeviation:       balanceDeviation,
	}, nil
}
