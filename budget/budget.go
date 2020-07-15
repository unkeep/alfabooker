package budget

import (
	"fmt"
	"net/http"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

// New creates budgest instance from google sheets source
func New(googleClient *http.Client, sheetID string) (*Budgets, error) {
	sheetsSrv, err := sheets.New(googleClient)
	if err != nil {
		return nil, fmt.Errorf("sheets.New: %w", err)
	}

	return &Budgets{
		sheetsSrv: sheetsSrv,
		sheetID:   sheetID,
	}, nil
}

type Budgets struct {
	sheetsSrv *sheets.Service
	sheetID   string
}

func (b *Budgets) List() ([]Budget, error) {
	readRange := "A1:D21"
	resp, err := b.sheetsSrv.Spreadsheets.Values.Get(b.sheetID, readRange).Do()
	if err != nil {
		return nil, err
	}

	result := make([]Budget, 0, len(resp.Values))
	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}

		name, _ := row[0].(string)

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

func (b *Budgets) IncreaseSpent(id string, value int) error {
	spentCell := "C" + id
	resp, err := b.sheetsSrv.Spreadsheets.Values.Get(b.sheetID, spentCell).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return fmt.Errorf("NotFound")
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

	call := b.sheetsSrv.Spreadsheets.Values.Update(b.sheetID, spentCell, update)
	call.ValueInputOption("RAW")

	_, err = call.Do()

	return err
}
