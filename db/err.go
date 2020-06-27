package db

import (
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrNotFound is an alias of mongo.ErrNoDocuments
var ErrNotFound = mongo.ErrNoDocuments
