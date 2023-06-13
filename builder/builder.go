package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var fileExifJson map[string]interface{}

type Configuration struct {
	DbType   string `json:"DbType"`
	Host     string `json:"Host"`
	Port     string `json:"Port"`
	DbUser   string `json:"DbUser"`
	DbPwd    string `json:"DbPwd"`
	DbName   string `json:"DbName"`
	FileColl string `json:"FileColl"`
	TreeColl string `json:"TreeColl"`
}

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

func init() {
	path = flag.String("path", "/home/delimp/Documents/project-phi", "full path")
}

func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashBytes := hash.Sum(nil)
	hashValue := hex.EncodeToString(hashBytes)

	return hashValue, nil
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

func main() {
	// Capture and string-ify path
	flag.Parse()
	basePath := *path

	// Read the directory
	err := readDirectory(basePath)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		os.Exit(1)
	}
}

func readDirectory(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		return nil
	}

	for _, file := range files {
		filePath := path + "/" + file.Name()

		if file.IsDir() {
			if err := readDirectory(filePath); err != nil {
				fmt.Printf("Error reading directory %s: %s\n", filePath, err)
			} else {
				fileValue := filePath
				fsName := file.Name()
				rawSize := file.Size()
				fsMode := file.Mode()
				fsModTime := file.ModTime()
				isDirectory := file.IsDir()
				fsSys := file.Sys()

				fmt.Println("Source File:", fileValue)
				fmt.Println("File Name:", fsName)
				fmt.Println("File Size:", rawSize)
				fmt.Println("File Mode:", fsMode)
				fmt.Println("File ModTime:", fsModTime)
				fmt.Println("File IsDir:", isDirectory)
				fmt.Println("File Sys:", fsSys)
			}

		} else {
			fileValue := filePath
			fsName := file.Name()
			rawSize := file.Size()
			fsMode := file.Mode()
			fsModTime := file.ModTime()
			isDirectory := file.IsDir()
			fsSys := file.Sys()

			fmt.Println("Source File:", fileValue)
			fmt.Println("File Name:", fsName)
			fmt.Println("File Size:", rawSize)
			fmt.Println("File Mode:", fsMode)
			fmt.Println("File ModTime:", fsModTime)
			fmt.Println("File IsDir:", isDirectory)
			fmt.Println("File Sys:", fsSys)
		}
	}

	return nil
}
