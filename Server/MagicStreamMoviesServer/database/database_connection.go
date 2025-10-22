package database 

import (
	"fmt"
	"log"
	"os"
	"time"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"
)

func Connect() *mongo.Client {
    if err := godotenv.Load(".env"); err != nil {
        log.Println("Warning: unable to find .env file")
    }

    MongoDb := os.Getenv("MONGODB_URI")
    if MongoDb == "" {
        log.Fatal("MONGODB_URI not set!")
    }

    clientOptions := options.Client().ApplyURI(MongoDb)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal(err)
    }

    return client
}


//var Client *mongo.Client = DBInstance()

func OpenCollection(collectionName string, client *mongo.Client) *mongo.Collection {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: unable to find .env file")
	}

	databaseName := os.Getenv("DATABASE_NAME")

	fmt.Println("DATABASE_NAME: ", databaseName)

	collection := client.Database(databaseName).Collection(collectionName)

	if collection == nil {
		return nil
	}
	return collection

}