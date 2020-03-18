package mongo

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
	ID       string `bson:"_id"`
	Time     time.Time
	Amount   float64
	Balance  float64
	Success  bool
	RawText  string
	Category string
}

// GetOperationCollection contructs a OperationCollection
func GetOperationCollection(cli *Client) *OperationCollection {
	return &OperationCollection{c: cli.database.Collection("operations")}
}

// OperationCollection priovides operations access methods
type OperationCollection struct {
	c *mongo.Collection
}

// ErrNotFound is an alias of mongo.ErrNoDocuments
var ErrNotFound = mongo.ErrNoDocuments

// GetOne gets an operation with the given ID
func (c *OperationCollection) GetOne(ctx context.Context, id string) (Operation, error) {
	filter := bson.M{"_id": id}
	res := c.c.FindOne(ctx, filter)
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
func (c *OperationCollection) Save(ctx context.Context, op Operation) error {
	filter := bson.M{"_id": op.ID}
	upd := bson.M{"$set": op}
	upsert := true
	opts := &options.UpdateOptions{Upsert: &upsert}

	res, err := c.c.UpdateOne(ctx, filter, upd, opts)

	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", *res)
	return nil
}
