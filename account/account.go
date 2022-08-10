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

func NewDomain(client *http.Client) (*Domain, error) {
	// nolint: staticcheck
	gmailSrv, err := gmail.New(client)
	if err != nil {
		return nil, fmt.Errorf("gmail.NewDomain: %w", err)
	}

	acc := &Domain{
		gmailSrv:  gmailSrv,
		amountRE:  regexp.MustCompile(`Сумма:(?:.*\()?([0-9]*\.?[0-9]*)\sBYN\)?`),
		balanceRE: regexp.MustCompile(`Balance:\s([0-9]*\.?[0-9]*)\sGEL`),
	}

	return acc, nil
}

type Domain struct {
	gmailSrv  *gmail.Service
	lastMsgID string
	amountRE  *regexp.Regexp
	balanceRE *regexp.Regexp
}

func (d *Domain) GetOperations(ctx context.Context, ops chan<- Operation) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			op, exist, err := d.getLastOperation()
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

func (d *Domain) getLastOperation() (Operation, bool, error) {
	listResp, err := d.gmailSrv.Users.Messages.List("me").Q("label:tbc-sms").MaxResults(1).Do()

	if err != nil {
		return Operation{}, false, fmt.Errorf("Messages.List: %w", err)
	}

	if len(listResp.Messages) == 0 {
		return Operation{}, false, nil
	}

	msgID := listResp.Messages[0].Id
	if msgID == d.lastMsgID {
		return Operation{}, false, nil
	}

	getResp, err := d.gmailSrv.Users.Messages.Get("me", msgID).Fields("payload").Do()
	if err != nil {
		return Operation{}, false, fmt.Errorf("Messages.Get(%s): %w", msgID, err)
	}

	data, err := base64.URLEncoding.DecodeString(getResp.Payload.Body.Data)
	if err != nil {
		return Operation{}, false, err
	}

	d.lastMsgID = msgID

	op, err := d.newOperation(msgID, data)
	if err != nil {
		return Operation{}, false, fmt.Errorf("newOperation(id: %s, data: %s): %w", msgID, string(data), err)
	}

	return op, true, nil
}

func (d *Domain) newOperation(id string, rawMsg []byte) (Operation, error) {
	// amount, err := d.parseAmount(rawMsg)
	// if err != nil {
	// 	return Operation{}, fmt.Errorf("parseAmount: %w", err)
	// }

	balance, err := d.parseBalance(rawMsg)
	if err != nil {
		return Operation{}, fmt.Errorf("parseBalance: %w", err)
	}

	// amountSign := 1.0
	// if isTransferOut(rawMsg) {
	// 	amountSign = -1
	// }

	return Operation{
		ID: id,
		// Amount:      amount * amountSign,
		Balance: balance,
		// Success:     parseSuccess(rawMsg),
		// Description: parseDescription(rawMsg),
		RawText: string(rawMsg),
	}, nil
}

func (d *Domain) parseAmount(body []byte) (float64, error) {
	res := d.amountRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, fmt.Errorf("unable to parse amount")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func (d *Domain) parseBalance(body []byte) (float64, error) {
	res := d.balanceRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, fmt.Errorf("unable to parse balance")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func parseDescription(body []byte) string {
	lines := strings.Split(string(body), "\n")
	if len(lines) < 3 {
		return ""
	}

	return strings.TrimSpace(lines[len(lines)-3])
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
