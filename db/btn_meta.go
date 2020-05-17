package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
)

// BtnMeta tg btn meta
type BtnMeta struct {
	ActionType   string
	OperationID  string
	CategotyID   string
	CategotyName string
	ChatID       int64
}

// BtnMetaRepo priovides bt metas access methods
type BtnMetaRepo struct {
	c *mongo.Collection
}

func getBtnMetaRepo(mngDB *mongo.Database) *BtnMetaRepo {
	return &BtnMetaRepo{c: mngDB.Collection("btnmeta")}
}

// GetOne gets btnBeta by id
func (r *BtnMetaRepo) GetOne(ctx context.Context, id string) (BtnMeta, error) {
	var btn BtnMeta
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return btn, err
	}
	filter := bson.M{"_id": objID}
	res := r.c.FindOne(ctx, filter)
	if res.Err() != nil {
		return btn, res.Err()
	}

	if err := res.Decode(&btn); err != nil {
		return btn, err
	}

	return btn, nil
}

// AddBatch adds btns meta batch
func (r *BtnMetaRepo) AddBatch(ctx context.Context, batch []BtnMeta) ([]string, error) {
	docs := make([]interface{}, 0, len(batch))
	for _, btn := range batch {
		docs = append(docs, btn)
	}

	res, err := r.c.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(res.InsertedIDs))
	for _, id := range res.InsertedIDs {
		ids = append(ids, id.(primitive.ObjectID).Hex())
	}

	return ids, nil
}
