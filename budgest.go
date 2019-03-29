package main

type Budget struct {
	ID     int
	Name   string
	Amount float64
	Spent  float64
}

type Budgets interface {
	List() ([]Budget, error)
	Update(b Budget) error
}
