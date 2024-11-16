package db

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const EnvPath = "./.env"

// ClientInterface Interface specifying the methods for the client
type ClientInterface interface {
	QueryCollection(collectionName string, conditions *bson.D) ([]interface{}, error)
	InsertDocument(collectionName string, document []byte) (string, error)
	DeleteDocument(collectionName string, conditions *bson.D) error
	EditDocument(collectionName string, conditions *bson.D, document []byte) error
}

// Client struct to hold the client connection and environment variables
type Client struct {
	dbClient *mongo.Client
	envVars  map[string]string
}

// QueryCollection Queries a named collection in the database based on some conditions.
// Returns an HTTP status code and an error.
func (client *Client) QueryCollection(collectionName string, conditions *bson.D, opts *options.FindOptions, results interface{}) (int, error) {
	// Accessing the collection
	collection := client.dbClient.Database(client.envVars["DB_NAME"]).Collection(collectionName)
	log.Printf("Accessed collection: %s", collection.Name())

	if conditions == nil {
		conditions = &bson.D{}
	}

	// Querying the collection
	cursor, err := collection.Find(context.Background(), conditions, opts)

	if err != nil {
		log.Printf("Error querying the collection: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("Error closing the cursor: %s", err.Error())
		}
	}(cursor, context.Background())

	// Decoding directly into a generic result
	if err = cursor.All(context.Background(), results); err != nil {
		log.Printf("Error decoding the results: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	// Checking if the cursor encountered any errors
	if err = cursor.Err(); err != nil {
		log.Printf("Error with the cursor: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	log.Printf("Query on %s OK!", collection.Name())
	return http.StatusOK, nil
}

// InsertDocument Inserts a new document into the named collection. The conditions parameter is used to check for uniqueness.
// Returns the ID of the inserted document, HTTP status code, and an error
func (client *Client) InsertDocument(collectionName string, conditions *bson.D, document interface{}) (interface{}, int, error) {
	collection := client.dbClient.Database(client.envVars["DB_NAME"]).Collection(collectionName)
	log.Printf("Accessed collection: %s", collection.Name())

	// Checking the insertion conditions
	// Mainly checking for uniqueness of the document
	if conditions != nil {
		cursor, err := collection.Find(context.Background(), conditions)

		if err != nil {
			log.Printf("Error querying the collection: %s", err.Error())
			return nil, http.StatusInternalServerError, err
		}

		defer func(cursor *mongo.Cursor, ctx context.Context) {
			err := cursor.Close(ctx)
			if err != nil {
				log.Printf("Error closing the cursor: %s", err.Error())
			}
		}(cursor, context.Background())

		if cursor.Next(context.Background()) {
			log.Printf("Document already exists in the collection!")
			return nil, http.StatusConflict, errors.New("document already exists in the collection")
		}
	}

	insertResult, err := collection.InsertOne(context.Background(), document)

	if err != nil {
		log.Printf("Error inserting the document: %s", err.Error())
		return nil, http.StatusInternalServerError, err
	}

	log.Printf("Inserted document with ID: %s", insertResult.InsertedID)
	return insertResult.InsertedID.(primitive.ObjectID), http.StatusCreated, nil
}

// DeleteDocument Deletes a document from the named collection based on the conditions provided.
// Returns the HTTP status code and an error.
func (client *Client) DeleteDocument(collectionName string, conditions *bson.D) (int, error) {
	collection := client.dbClient.Database(client.envVars["DB_NAME"]).Collection(collectionName)
	log.Printf("Accessed collection: %s", collection.Name())

	deletedResult, err := collection.DeleteOne(context.Background(), conditions)

	if err != nil {
		log.Printf("Error deleting the document: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	// Checking if the document was actually deleted
	if deletedResult.DeletedCount == 0 {
		log.Printf("Document not found in the collection!")
		return http.StatusBadRequest, errors.New("item not found, the ID might be incorrect")
	}

	return http.StatusOK, nil
}

// EditDocument Replaces a document in the named collection based on the conditions provided.
// Returns the HTTP status code and an error.
func (client *Client) EditDocument(collectionName string, conditions *bson.D, document interface{}) (int, error) {
	collection := client.dbClient.Database(client.envVars["DB_NAME"]).Collection(collectionName)
	log.Printf("Accessed collection: %s", collection.Name())

	replaceResult, err := collection.ReplaceOne(context.Background(), conditions, document)

	if err != nil {
		log.Printf("Error updating the document: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	// Verifying if the document was actually updated
	if replaceResult.ModifiedCount == 0 {
		log.Printf("Document not found in the collection!")
		return http.StatusBadRequest, errors.New("item not found, the ID might be incorrect or the item is the same")
	}

	return http.StatusOK, nil
}

// Function to read the environment variables from the .env file
func readEnv(envPath string) (map[string]string, error) {
	err := godotenv.Load(envPath)

	if err != nil {
		return nil, err
	}

	envVars := make(map[string]string)

	envVars["DB_URI"] = os.Getenv("DB_URI")
	envVars["DB_NAME"] = os.Getenv("DB_NAME")

	return envVars, nil
}

// PrepareClient Function to prepare the client connection
func PrepareClient() (*Client, error) {
	envVars, err := readEnv(EnvPath)

	if err != nil {
		log.Printf("Error reading the environment variables: %s", err.Error())
		return nil, err
	}

	// Set the server API to stable version 1
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)

	// Initialize the client apiOptions
	apiOptions := options.Client().ApplyURI(envVars["DB_URI"]).SetServerAPIOptions(serverAPIOptions)

	// Creating a new client and connecting to the server
	client, err := mongo.Connect(context.Background(), apiOptions)

	if err != nil {
		log.Printf("Error connecting to the server: %s", err.Error())
		return nil, err
	}

	// Ping the server to ensure the connection is established
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Printf("Error pinging the server: %s", err.Error())
		return nil, err
	}

	return &Client{
		dbClient: client,
		envVars:  envVars,
	}, nil
}

// Cleanup Function to clean up the client connection
func Cleanup(client *Client) {
	if err := client.dbClient.Disconnect(context.Background()); err != nil {
		log.Panicf("Error disconnecting from the server: %s", err.Error())
	}
}
