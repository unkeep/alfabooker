package mongo

import (
	"context"
	"testing"
	"time"
)

const mongoURI = "mongodb+srv://test:testtest@cluster0-z78de.mongodb.net/test?retryWrites=true&w=majority"

func TestUpdate(t *testing.T) {
	cli, err := GetClient(context.Background(), mongoURI)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.client.Disconnect(context.Background())

	coll := GetOperationCollection(cli)

	op := Operation{
		ID:       "test",
		Amount:   123.123,
		Balance:  500.123,
		Category: "test",
		RawText:  "bla bla",
		Success:  true,
		Time:     time.Now(),
	}

	if err := coll.Save(context.Background(), op); err != nil {
		t.Fatal(err)
	}
}
