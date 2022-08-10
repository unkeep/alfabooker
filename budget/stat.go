package budget

type Statistics struct {
	BudgetAmount float64 `json:"budget_amount"`

	BudgetStartedAt        int64   `json:"budget_started_at"`
	BudgetExpiresAt        int64   `json:"budget_expires_at"`
	BudgetDaysToExpiration float64 `json:"budget_days_to_expiration"`

	AccountBalance float64 `json:"account_balance"`
	CashBalance    float64 `json:"cash_balance"`
	TotalBalance   float64 `json:"total_balance"`

	EstimatedBalance float64 `json:"estimated_balance"`
	BalanceDeviation float64 `json:"balance_deviation"`
}
