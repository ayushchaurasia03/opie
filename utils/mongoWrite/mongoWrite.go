package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FileData struct {
	IsDirectory         bool   `json:"IsDirectory"`
	FileSizeRaw         int64  `json:"FileSizeRaw"`
	FileHash            string `json:"FileHash"`
	SourceFileHash      string `json:"SourceFileHash"`
	DirectoryHash       string `json:"DirectoryHash"`
	FileContent         []byte `json:"FileContent,omitempty"`
	FileInodeChangeDate string `json:"FileInodeChangeDate"`
	FileSize            string `json:"FileSize"`
	FileType            string `json:"FileType"`
	MIMEType            string `json:"MIMEType"`
	WordCount           int    `json:"WordCount"`
	FilePermissions     string `json:"FilePermissions"`
	LineCount           int    `json:"LineCount"`
	MIMEEncoding        string `json:"MIMEEncoding"`
	ExifToolVersion     string `json:"ExifToolVersion"`
	FileAccessDate      string `json:"FileAccessDate"`
	FileName            string `json:"FileName"`
	FileTypeExtension   string `json:"FileTypeExtension"`
	SourceFile          string `json:"SourceFile"`
	Directory           string `json:"Directory"`
	FileModifyDate      string `json:"FileModifyDate"`
	Newlines            string `json:"Newlines"`
}

type DirectoryData struct {
	IsDirectory       bool        `json:"IsDirectory"`
	SourceFile        string      `json:"SourceFile"`
	SourceFileHash    string      `json:"SourceFileHash"`
	Directory         string      `json:"Directory"`
	DirectoryHash     string      `json:"DirectoryHash"`
	FileModifyDate    time.Time   `json:"FileModifyDate"`
	FilePermissions   os.FileMode `json:"FilePermissions"`
	FileSizeRaw       int64       `json:"FileSizeRaw"`
	FileTypeExtension string      `json:"FileTypeExtension"`
	IsRoot            bool        `json:"IsRoot"`
	RootDirectoryName string      `json:"RootDirectoryName"`
	TotalSize         int64       `json:"TotalSize"`
	Directories       int         `json:"Directories"`
	Files             int         `json:"Files"`
}

func connectToMongoDB(dbType, host, port, dbUser, dbPwd, dbName, collectionName string) (*mongo.Client, *mongo.Collection, context.Context, error) {
	// Construct MongoDB connection URI
	mongodbURI := dbType + "://" + dbUser + ":" + dbPwd + "@" + host + ":" + port

	// Configure the client connection
	clientOptions := options.Client().ApplyURI(mongodbURI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Check if the connection was successful
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Access the specified database and collection
	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	// Create a context with a 15-second timeout
	ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)

	return client, collection, ctx, nil
}

func insertFileDataIntoMongoDB(data FileData, collection *mongo.Collection, ctx context.Context) error {
	insertResult, err := collection.InsertOne(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to insert file data into MongoDB FILEEE: %v", err)
	}

	fmt.Println("Inserted document ID:", insertResult.InsertedID)

	return nil
}

func insertDirectoryDataIntoMongoDB(data DirectoryData, collection *mongo.Collection, ctx context.Context) error {

	insertResult, err := collection.InsertOne(ctx, data)
	fmt.Println("Inserted document ID:", insertResult)

	if err != nil {
		return fmt.Errorf("failed to insert directory data into MongoDB DIRECTORYYYYYY: %v", err)
	}

	fmt.Println("Inserted document ID:", insertResult.InsertedID)

	return nil
}
