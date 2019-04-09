package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"
)

// Budget is a budget info
type Budget struct {
	ID       string
	Name     string
	Amount   int
	Spent    int
	SpentPct uint8
}

// Budgets is an budgets access interface
type Budgets interface {
	List() ([]Budget, error)
	IncreaseSpent(id string, value float64) error
	SetClient(client *http.Client) error
}

// GetBudgets creates budgest instance from google sheets source
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
	readRange := "A1:D14"
	resp, err := b.srv.Spreadsheets.Values.Get(b.sheetID, readRange).Do()
	if err != nil {
		return nil, err
	}

	result := make([]Budget, 0, len(resp.Values))
	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}

		name, _ := row[0].(string)
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}

		amountStr, _ := row[1].(string)
		amount, _ := strconv.Atoi(amountStr)
		if amount == 0 {
			continue
		}

		spentStr, _ := row[2].(string)
		spent, _ := strconv.Atoi(spentStr)

		pctStr, _ := row[3].(string)
		pct, _ := strconv.Atoi(pctStr)
		result = append(result, Budget{
			ID:       strconv.Itoa(i + 1),
			Name:     name,
			Amount:   amount,
			Spent:    spent,
			SpentPct: uint8(pct),
		})
	}

	return result, nil
}

func (b *budgetsImpl) IncreaseSpent(id string, value float64) error {
	spentCell := "C" + id
	resp, err := b.srv.Spreadsheets.Values.Get(b.sheetID, spentCell).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return errors.New("NotFound")
	}

	spentValueStr, ok := resp.Values[0][0].(string)
	if !ok {
		return fmt.Errorf("unable to converto to int: %v", resp.Values[0][0])
	}

	spentValue, err := strconv.Atoi(spentValueStr)
	if err != nil {
		return err
	}

	spentValue += int(value)

	update := &sheets.ValueRange{}
	update.MajorDimension = "ROWS"
	update.Values = [][]interface{}{{spentValue}}

	call := b.srv.Spreadsheets.Values.Update(b.sheetID, spentCell, update)
	call.ValueInputOption("RAW")

	_, err = call.Do()

	return err
}

func (b *budgetsImpl) SetClient(client *http.Client) error {
	srv, err := sheets.New(client)
	if err != nil {
		return err
	}
	b.srv = srv

	return nil
}
