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

func GetAccount(emal string, pass string) (Account, error) {
	log.Println("Connecting to the imap.gmail.com:993...")

	// Connect to server
	client, err := imapClient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return nil, err
	}
	log.Println("Connected")

	// Login
	if err := client.Login(emal, pass); err != nil {
		return nil, err
	}
	log.Println("Logged in")

	return &accountImpl{
		client:   client,
		amountRE: regexp.MustCompile(`Сумма:(?:.*\()?([0-9]*\.?[0-9]*)\sBYN\)?`),
	}, nil
}

type accountImpl struct {
	client     *imapClient.Client
	lastMsgNum uint32
	amountRE   *regexp.Regexp
}

func (acc *accountImpl) GetLastOperation() (Operation, error) {
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
