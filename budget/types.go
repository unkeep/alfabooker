package budget

// Budget is a budget info
type Budget struct {
	ID       string
	Name     string
	Amount   int
	Spent    int
	SpentPct uint8
}
