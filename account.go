package main

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// errOperationNotFound should be returned when new operations are not found
var errOperationNotFound = errors.New("errOperationNotFound")

// Account is an account interface
type Account interface {
	GetOperationsChan() <-chan Operation
	SetClient(client *http.Client) error
}

// GetAccount creates an account
func GetAccount() (Account, error) {
	acc := &accountImpl{
		opChan:    make(chan Operation),
		amountRE:  regexp.MustCompile(`Сумма:(?:.*\()?([0-9]*\.?[0-9]*)\sBYN\)?`),
		balanceRE: regexp.MustCompile(`Остаток:([0-9]*\.?[0-9]*)\sBYN`),
	}

	return acc, nil
}

type accountImpl struct {
	opChan    chan Operation
	srv       *gmail.Service
	lastMsgID string
	amountRE  *regexp.Regexp
	balanceRE *regexp.Regexp
}

func (acc *accountImpl) GetOperationsChan() <-chan Operation {
	return acc.opChan
}

func (acc *accountImpl) SetClient(client *http.Client) error {
	srv, err := gmail.New(client)
	if err != nil {
		return err
	}
	acc.srv = srv

	go acc.pollEmails()

	return nil
}

func (acc *accountImpl) pollEmails() {
	for {
		op, err := acc.getLastOperation()
		if err == nil {
			acc.opChan <- op
			continue
		}

		if err != errOperationNotFound {
			// TODO: handle err
			log.Println(err)
		}

		time.Sleep(time.Second * 5)
	}
}

func (acc *accountImpl) getLastOperation() (Operation, error) {
	listResp, err := acc.srv.Users.Messages.List("me").Q("label:alfa-bank").MaxResults(1).Do()

	if err != nil {
		return Operation{}, err
	}

	if len(listResp.Messages) == 0 {
		return Operation{}, errOperationNotFound
	}

	id := listResp.Messages[0].Id
	if id == acc.lastMsgID {
		return Operation{}, errOperationNotFound
	}

	getResp, err := acc.srv.Users.Messages.Get("me", id).Fields("payload").Do()
	if err != nil {
		return Operation{}, err
	}

	data, err := base64.URLEncoding.DecodeString(getResp.Payload.Body.Data)
	if err != nil {
		return Operation{}, err
	}

	acc.lastMsgID = id

	amount, err := acc.parseAmount(data)
	if err != nil {
		log.Println(err)
	}

	balance, err := acc.parseBalance(data)
	if err != nil {
		log.Println(err)
	}

	return Operation{
		ID:          id,
		Amount:      amount,
		Balance:     balance,
		Type:        acc.parseType(data),
		Success:     acc.parseSuccess(data),
		Description: string(data),
	}, nil
}

func (acc *accountImpl) parseAmount(body []byte) (float64, error) {
	res := acc.amountRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, errors.New("unable to parse amount")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func (acc *accountImpl) parseBalance(body []byte) (float64, error) {
	res := acc.balanceRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, errors.New("unable to parse balance")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func (acc *accountImpl) parseType(body []byte) OperationType {
	str := string(body)
	if strings.Contains(str, "Оплата товаров/услуг") || strings.Contains(str, "Перевод (Списание)") {
		return DecreasingOperation
	}

	if strings.Contains(str, "Перевод (Поступление)") {
		return IncreasingOperation
	}

	return UndefinedOperation
}

func (acc *accountImpl) parseSuccess(body []byte) bool {
	return strings.Contains(string(body), "Успешно")
}
