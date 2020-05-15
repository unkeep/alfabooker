package db

import (
	"context"

	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

func GetClient(ctx context.Context, mongoURI string) (*Client, error) {
	connStr, err := connstring.Parse(mongoURI)
	if err != nil {
		return nil, err
	}

	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	return &Client{client: cli, database: cli.Database(connStr.Database)}, nil
}
