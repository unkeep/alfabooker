package account

// Operation is an operation info
type Operation struct {
	ID          string
	Amount      float64
	Balance     float64
	Success     bool
	Description string
	RawText     string
}
