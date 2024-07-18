package database

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbConfig struct {
	ConnectionURL string
	Database      string
}

func GetConnection(ctx context.Context, config MongodbConfig) (*mongo.Client, *mongo.Database, error) {
	server_api := options.ServerAPI(options.ServerAPIVersion1)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.ConnectionURL).SetServerAPIOptions(server_api))
	if err != nil {
		panic(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	database := client.Database(config.Database)
	return client, database, nil
}
