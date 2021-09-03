package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

type Repo struct {
	Tokens     *TokensRepo
	Budget     *BudgetRepo
}

func (r *Repo) Close() {

}

func GetRepo(ctx context.Context, mongoURI string) (*Repo, error) {
	connStr, err := connstring.Parse(mongoURI)
	if err != nil {
		return nil, err
	}

	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		disCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		cli.Disconnect(disCtx)
	}()

	db := cli.Database(connStr.Database)

	return &Repo{
		Tokens:     getTokensRepo(db),
		Budget:     getBudgetRepo(db),
	}, nil
}
