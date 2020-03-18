package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
)

// BtnMeta tg btn meta
type BtnMeta struct {
	ActionType  string
	OperationID string
	CategotyID  string
}

// BtnMetaCollection priovides bt metas access methods
type BtnMetaCollection struct {
	c *mongo.Collection
}

// GetBtnMetaCollection contructs a BtnMetaCollection
func GetBtnMetaCollection(cli *Client) *BtnMetaCollection {
	return &BtnMetaCollection{c: cli.database.Collection("btnmeta")}
}

// GetOne gets btnBeta by id
func (c *BtnMetaCollection) GetOne(ctx context.Context, id string) (BtnMeta, error) {
	var btn BtnMeta
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return btn, err
	}
	filter := bson.M{"_id": objID}
	res := c.c.FindOne(ctx, filter)
	if res.Err() != nil {
		return btn, res.Err()
	}

	if err := res.Decode(&btn); err != nil {
		return btn, err
	}

	return btn, nil
}

// AddBatch adds btns meta batch
func (c *BtnMetaCollection) AddBatch(ctx context.Context, batch []BtnMeta) ([]string, error) {
	docs := make([]interface{}, 0, len(batch))
	for _, btn := range batch {
		docs = append(docs, btn)
	}

	res, err := c.c.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(res.InsertedIDs))
	for _, id := range res.InsertedIDs {
		ids = append(ids, id.(primitive.ObjectID).Hex())
	}

	return ids, nil
}
