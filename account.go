package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"

	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

var ErrOperationNotFound = errors.New("ErrOperationNotFound")

type Account interface {
	GetLastOperation() (Operation, error)
	Logout()
}

func GetAccount(email string, pass string) (Account, error) {
	acc := &accountImpl{
		email:    email,
		pass:     pass,
		amountRE: regexp.MustCompile(`Сумма:(?:.*\()?([0-9]*\.?[0-9]*)\sBYN\)?`),
	}

	err := acc.connect()

	return acc, err
}

type accountImpl struct {
	email      string
	pass       string
	client     *imapClient.Client
	lastMsgNum uint32
	amountRE   *regexp.Regexp
}

func (acc *accountImpl) GetLastOperation() (Operation, error) {
	op, err := acc.getLastOperation()

	// reconnect and retry if nessesary
	if err != nil && err.Error() == "imap: connection closed" {
		if acc.connect(); err != nil {
			return Operation{}, err
		}
		return acc.getLastOperation()
	}

	return op, err
}

func (acc *accountImpl) getLastOperation() (Operation, error) {
	mbox, err := acc.client.Select("alfa-bank", true)
	if err != nil {
		return Operation{}, err
	}

	msgNum := mbox.Messages

	if msgNum == acc.lastMsgNum {
		return Operation{}, ErrOperationNotFound
	}

	op, err := acc.getOperation(msgNum)
	if err == nil {
		acc.lastMsgNum = msgNum
	}

	return op, err
}

func (acc *accountImpl) parseAmount(body []byte) (float64, error) {
	res := acc.amountRE.FindSubmatch(body)
	if len(res) != 2 {
		return 0, errors.New("unable to parse amount")
	}

	return strconv.ParseFloat(string(res[1]), 64)
}

func (acc *accountImpl) getOperation(num uint32) (Operation, error) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(num)

	section := &imap.BodySectionName{}
	fetchItems := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	if err := acc.client.Fetch(seqset, fetchItems, messages); err != nil {
		return Operation{}, nil
	}

	msg := <-messages

	if msg == nil {
		return Operation{}, ErrOperationNotFound
	}

	body := msg.GetBody(section)

	// Create a new mail reader
	mr, err := mail.CreateReader(body)
	if err != nil {
		return Operation{}, err
	}

	var bodyBytes []byte
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return Operation{}, err
		}

		switch p.Header.(type) {
		case mail.TextHeader:
			bodyBytes, err = ioutil.ReadAll(p.Body)
			if err != nil {
				return Operation{}, err
			}
		}
	}

	amount, err := acc.parseAmount(bodyBytes)
	if err != nil {
		log.Println("Error: ", err)
	}

	return Operation{
		ID:          strconv.Itoa(int(num)),
		Description: string(bodyBytes),
		Amount:      amount,
	}, nil
}

func (acc *accountImpl) Logout() {
	acc.client.Logout()
}

func (acc *accountImpl) connect() error {
	log.Println("Connecting to the imap.gmail.com:993...")

	if acc.client != nil {
		acc.client.Close()
	}

	// Connect to server
	client, err := imapClient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return err
	}
	log.Println("Connected")

	acc.client = client

	// Login
	if err := client.Login(acc.email, acc.pass); err != nil {
		return err
	}
	log.Println("Logged in")

	return nil
}
