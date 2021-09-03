package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Budget struct {
	ID   string `bson:"_id"`
	Amount float64
	StartedAt int64
	ExpiresAt int64
	Balance float64
}

const budgetID = "budget"

func getBudgetRepo(mngDB *mongo.Database) *BudgetRepo {
	return &BudgetRepo{c: mngDB.Collection("budget")}
}

type BudgetRepo struct {
	c *mongo.Collection
}

func (r *BudgetRepo) Get(ctx context.Context) (Budget, error) {
	filter := bson.M{"_id": budgetID}
	res := r.c.FindOne(ctx, filter)
	var b Budget
	if res.Err() != nil {
		return b, res.Err()
	}

	if err := res.Decode(&b); err != nil {
		return b, err
	}

	return b, nil
}

func (r *BudgetRepo) Save(ctx context.Context, b Budget) error {
	filter := bson.M{"_id": b.ID}
	upd := bson.M{"$set": b}
	upsert := true
	opts := &options.UpdateOptions{Upsert: &upsert}

	_, err := r.c.UpdateOne(ctx, filter, upd, opts)

	if err != nil {
		return err
	}
	return nil
}
