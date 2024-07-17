package handlers

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type Handlers struct {
	context  context.Context
	database *mongo.Database
}

func NewHandlers(context context.Context, database *mongo.Database) *Handlers {
	return &Handlers{
		database: database,
		context:  context,
	}
}
