package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Operation is struct of an operation stored in db
type Operation struct {
	ID          string `bson:"_id"`
	Time        time.Time
	Amount      float64
	Balance     float64
	Success     bool
	RawText     string
	Description string
	Category    string
}

func getOperationsRepo(mngDB *mongo.Database) *OperationsRepo {
	return &OperationsRepo{c: mngDB.Collection("operations")}
}

// OperationsRepo priovides operations access methods
type OperationsRepo struct {
	c *mongo.Collection
}

// ErrNotFound is an alias of mongo.ErrNoDocuments
var ErrNotFound = mongo.ErrNoDocuments

// GetOne gets an operation with the given ID
func (r *OperationsRepo) GetOne(ctx context.Context, id string) (Operation, error) {
	filter := bson.M{"_id": id}
	res := r.c.FindOne(ctx, filter)
	var op Operation
	if res.Err() != nil {
		return op, res.Err()
	}

	if err := res.Decode(&op); err != nil {
		return op, err
	}

	return op, nil
}

// Save saves an operation
func (r *OperationsRepo) Save(ctx context.Context, op Operation) error {
	filter := bson.M{"_id": op.ID}
	upd := bson.M{"$set": op}
	upsert := true
	opts := &options.UpdateOptions{Upsert: &upsert}

	res, err := r.c.UpdateOne(ctx, filter, upd, opts)

	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", *res)
	return nil
}
