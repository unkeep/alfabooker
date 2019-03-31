package main

import (
	"log"
	"net/http"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

type Budget struct {
	ID   string
	Name string
}

type Budgets interface {
	List() ([]Budget, error)
	IncreaseSpent(id int, value float64) error
	SetClient(client *http.Client) error
}

func GetBudgets(sheetID string) (Budgets, error) {
	return &budgetsImpl{
		sheetID: sheetID,
	}, nil
}

type budgetsImpl struct {
	srv     *sheets.Service
	sheetID string
}

func (b *budgetsImpl) List() ([]Budget, error) {
	readRange := "A1:A15"
	resp, err := b.srv.Spreadsheets.Values.Get(b.sheetID, readRange).Do()
	if err != nil {
		return nil, err
	}

	result := make([]Budget, 0, len(resp.Values))
	for i, row := range resp.Values {
		value := row[0]
		text, ok := value.(string)
		if ok && text != "" {
			result = append(result, Budget{
				ID:   strconv.Itoa(i),
				Name: text,
			})
		}
	}

	return result, nil
}

func (b *budgetsImpl) IncreaseSpent(id int, value float64) error {
	log.Println("IncreaseSpent: ", id, " - ", value)
	return nil
}

func (b *budgetsImpl) SetClient(client *http.Client) error {
	srv, err := sheets.New(client)
	if err != nil {
		return err
	}
	b.srv = srv

	return nil
}
