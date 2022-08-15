package budget

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/unkeep/alfabooker/db"
)

type Domain struct {
	budgetRepo *db.BudgetRepo
	balanceRE  *regexp.Regexp
}

func NewDomain(repo *db.BudgetRepo) *Domain {
	return &Domain{
		budgetRepo: repo,
		balanceRE:  regexp.MustCompile(`Balance:\s([0-9]*\.?[0-9]*)\sGEL`),
	}
}

func (d *Domain) GetStat(ctx context.Context) (*Statistics, error) {
	b, err := d.budgetRepo.Get(ctx)
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

func (d *Domain) UpdateAccountBalanceFromSMS(ctx context.Context, sms string) error {
	balance, err := d.parseBalanceFromSMS(sms)
	if err != nil {
		return fmt.Errorf("parseBalanceFromSMS: %w", err)
	}

	b, err := d.budgetRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("BudgetRepo.Get: %w", err)
	}

	b.Balance = balance

	if err := d.budgetRepo.Save(ctx, b); err != nil {
		return fmt.Errorf("BudgetRepo.Save: %w", err)
	}

	return nil
}

func (d *Domain) parseBalanceFromSMS(sms string) (float64, error) {
	res := d.balanceRE.FindSubmatch([]byte(sms))
	if len(res) != 2 {
		return 0, fmt.Errorf("unable to parse balance")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}
