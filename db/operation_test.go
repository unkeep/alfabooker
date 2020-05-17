package db

import (
	"context"
	"reflect"
	"testing"
	"time"
)

const mongoURI = "mongodb+srv://test:testtest@cluster0-z78de.mongodb.net/test?retryWrites=true&w=majority"

func TestOperationUpdateGet(t *testing.T) {
	repo, err := GetRepo(context.Background(), mongoURI)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)

	op := Operation{
		ID:       "test",
		Amount:   123.123,
		Balance:  500.123,
		Category: "test",
		RawText:  "bla bla",
		Success:  true,
		Time:     now,
	}

	if err := repo.Operations.Save(context.Background(), op); err != nil {
		t.Fatal(err)
	}

	gotOp, err := repo.Operations.GetOne(context.Background(), op.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotOp, op) {
		t.Error("got != expected")
		t.Logf("got:      %+v", gotOp)
		t.Logf("expected: %+v", op)
	}
}
