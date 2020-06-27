package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Token struct {
	ID   string `bson:"_id"`
	Time time.Time
	Data []byte
}

func getTokensRepo(mngDB *mongo.Database) *TokensRepo {
	return &TokensRepo{c: mngDB.Collection("tokens")}
}

// TokensRepo priovides operations access methods
type TokensRepo struct {
	c *mongo.Collection
}

// GetOne gets a token with the given ID
func (r *TokensRepo) GetOne(ctx context.Context, id string) (Token, error) {
	filter := bson.M{"_id": id}
	res := r.c.FindOne(ctx, filter)
	var op Token
	if res.Err() != nil {
		return op, res.Err()
	}

	if err := res.Decode(&op); err != nil {
		return op, err
	}

	return op, nil
}

// Save saves a token
func (r *TokensRepo) Save(ctx context.Context, t Token) error {
	filter := bson.M{"_id": t.ID}
	upd := bson.M{"$set": t}
	upsert := true
	opts := &options.UpdateOptions{Upsert: &upsert}

	_, err := r.c.UpdateOne(ctx, filter, upd, opts)

	if err != nil {
		return err
	}
	return nil
}
