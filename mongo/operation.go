package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Operation -
type Operation struct {
	ID       string `bson:"_id"`
	Time     time.Time
	Amount   float64
	Balance  float64
	Success  bool
	RawText  string
	Category string
}

func GetOperationCollection(cli *Client) *OperationCollection {
	return &OperationCollection{c: cli.database.Collection("operations")}
}

type OperationCollection struct {
	c *mongo.Collection
}

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
