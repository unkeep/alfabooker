package db

import (
	"context"
	"reflect"
	"testing"
)

func TestBtnMetaAddGet(t *testing.T) {
	repo, err := GetRepo(context.Background(), mongoURI)
	if err != nil {
		t.Fatal(err)
	}

	batch := []BtnMeta{
		{
			ActionType:   "type1",
			CategotyID:   "cat1",
			OperationID:  "id1",
			CategotyName: "name1",
			ChatID:       11111,
		},
		{
			ActionType:   "type2",
			CategotyID:   "cat2",
			OperationID:  "id2",
			CategotyName: "name2",
			ChatID:       22222,
		},
	}

	ids, err := repo.BtnMetas.AddBatch(context.Background(), batch)
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 2 {
		t.Errorf("got ids %v, expected 2", ids)
	}

	got := make([]BtnMeta, 0, len(ids))
	for _, id := range ids {
		btn, err := repo.BtnMetas.GetOne(context.Background(), id)
		if err != nil {
			t.Error(err)
		}
		got = append(got, btn)
	}

	if !reflect.DeepEqual(got, batch) {
		t.Error("got != expected")
		t.Logf("got:      %+v", got)
		t.Logf("expected: %+v", batch)
	}
}
