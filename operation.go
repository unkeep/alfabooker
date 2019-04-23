package main

// OperationType is an operation type
type OperationType uint8

const (
	// UndefinedOperation is an udefined operation type
	UndefinedOperation OperationType = iota
	// IncreasingOperation is a increasing operation type
	IncreasingOperation = iota
	// DecreasingOperation is a decreasing operation type
	DecreasingOperation = iota
)

// Operation is an operation info
type Operation struct {
	ID          string
	Amount      float64
	Balance     float64
	Type        OperationType
	Success     bool
	Description string
}
