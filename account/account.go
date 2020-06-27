package account

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

func New(client *http.Client) (*Account, error) {
	gmailSrv, err := gmail.New(client)
	if err != nil {
		return nil, fmt.Errorf("gmail.New: %w", err)
	}

	acc := &Account{
		gmailSrv:  gmailSrv,
		amountRE:  regexp.MustCompile(`Сумма:(?:.*\()?([0-9]*\.?[0-9]*)\sBYN\)?`),
		balanceRE: regexp.MustCompile(`Остаток:([0-9]*\.?[0-9]*)\sBYN`),
	}

	return acc, nil
}

type Account struct {
	gmailSrv  *gmail.Service
	lastMsgID string
	amountRE  *regexp.Regexp
	balanceRE *regexp.Regexp
}

func (acc *Account) GetOperations(ctx context.Context, ops chan<- Operation) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			op, exist, err := acc.getLastOperation()
			if err != nil {
				log.Println(err)
				continue
			}
			if exist {
				ops <- op
			}
		}
	}
}

func (acc *Account) getLastOperation() (Operation, bool, error) {
	listResp, err := acc.gmailSrv.Users.Messages.List("me").Q("label:alfa-bank").MaxResults(1).Do()

	if err != nil {
		return Operation{}, false, fmt.Errorf("Messages.List: %w", err)
	}

	if len(listResp.Messages) == 0 {
		return Operation{}, false, nil
	}

	msgID := listResp.Messages[0].Id
	if msgID == acc.lastMsgID {
		return Operation{}, false, nil
	}

	getResp, err := acc.gmailSrv.Users.Messages.Get("me", msgID).Fields("payload").Do()
	if err != nil {
		return Operation{}, false, fmt.Errorf("Messages.Get(%s): %w", msgID, err)
	}

	data, err := base64.URLEncoding.DecodeString(getResp.Payload.Body.Data)
	if err != nil {
		return Operation{}, false, err
	}

	acc.lastMsgID = msgID

	op, err := acc.newOperation(msgID, data)
	if err != nil {
		return Operation{}, false, fmt.Errorf("newOperation(id: %s, data: %s): %w", msgID, string(data), err)
	}

	return op, true, nil
}

func (acc *Account) newOperation(id string, rawMsg []byte) (Operation, error) {
	amount, err := acc.parseAmount(rawMsg)
	if err != nil {
		return Operation{}, fmt.Errorf("parseAmount: %w", err)
	}

	balance, err := acc.parseBalance(rawMsg)
	if err != nil {
		return Operation{}, fmt.Errorf("parseBalance: %w", err)
	}

	amountSign := 1.0
	if isTransferOut(rawMsg) {
		amountSign = -1
	}

	return Operation{
		ID:          id,
		Amount:      amount * amountSign,
		Balance:     balance,
		Success:     parseSuccess(rawMsg),
		Description: string(rawMsg),
		RawText:     string(rawMsg),
	}, nil
}

func (acc *Account) parseAmount(body []byte) (float64, error) {
	res := acc.amountRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, fmt.Errorf("unable to parse amount")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func (acc *Account) parseBalance(body []byte) (float64, error) {
	res := acc.balanceRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, fmt.Errorf("unable to parse balance")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func isTransferOut(body []byte) bool {
	str := string(body)
	if strings.Contains(str, "Оплата товаров/услуг") || strings.Contains(str, "Перевод (Списание)") {
		return true
	}

	return false
}

func parseSuccess(body []byte) bool {
	return strings.Contains(string(body), "Успешно")
}
