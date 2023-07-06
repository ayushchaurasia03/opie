package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var config *string
var targetName *string
var fieldName *string
var substring *string
var fileCollection *mongo.Collection
var workerCount = 0
var workerPool = make(chan struct{}, workerCount)

func main() {
	flag.Parse()
}

func init() {
	// Read the configuration file
	config, err := readConfig("conf.json")
	if err != nil {
		fmt.Printf("Failed to read configuration file: %v\n", err)
		return
	}

	// targetName = flag.String("targetName", "", "The _id of the target document")
	fieldName = flag.String("fieldName", "", "The fieldname of the target field")
	substring = flag.String("substring", "", "The substring to search for")

	// Update the workerCount value
	workerCount = config.MaxGoroutines
	workerPool = make(chan struct{}, workerCount)
}

func countDocumentsWithSubstring(fieldName, substring string) int64 {
	// Set up MongoDB client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Access the collection and perform the count operation
	collection := client.Database("your-database-name").Collection("your-collection-name")

	filter := bson.D{{fieldName, bson.M{
		"$regex": substring,
	}}}

	countOptions := options.Count().SetHint(fieldName)
	count, err := collection.CountDocuments(context.Background(), filter, countOptions)
	if err != nil {
		log.Fatal(err)
	}

	return count
}

func sumFieldWithSubstring(fieldName, substring string) (float64, error) {
	// Set up MongoDB client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return 0, err
	}

	// Access the collection and perform the aggregation
	collection := client.Database("your-database-name").Collection("your-collection-name")

	pipeline := bson.A{
		bson.M{
			"$match": bson.M{
				fieldName: bson.M{
					"$regex": substring,
				},
			},
		},
		bson.M{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$FileSizeRaw"},
			},
		},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.Background())

	if cursor.Next(context.Background()) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			return 0, err
		}

		total := result["total"].(float64)
		return total, nil
	}

	// No matching documents found
	return 0, nil
}
