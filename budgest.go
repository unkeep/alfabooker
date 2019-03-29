package main

import "errors"

type Budget struct {
	ID   string
	Name string
}

type Budgets interface {
	List() ([]Budget, error)
	IncreaseSpent(id int, value float64) error
}

func GetBudgets() (Budgets, error) {
	return &budgetsImpl{}, nil
}

type budgetsImpl struct {
}

func (b *budgetsImpl) List() ([]Budget, error) {
	return []Budget{}, nil
}

func (b *budgetsImpl) IncreaseSpent(id int, value float64) error {
	return errors.New("notImplemented")
}
