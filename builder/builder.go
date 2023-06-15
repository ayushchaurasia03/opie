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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var fileCollection *mongo.Collection

type FileData struct {
	IsDirectory     bool   `json:"IsDirectory"`
	FileSizeRaw     int64  `json:"FileSizeRaw"`
	FileHash        string `json:"FileHash"`
	SourceFileHash  string `json:"SourceFileHash"`
	DirectoryHash   string `json:"DirectoryHash"`
	FilePermissions string `json:"FilePermissions"` // Updated to string type
	FileName        string `json:"FileName"`
	SourceFile      string `json:"SourceFile"`
	Directory       string `json:"Directory"`
	FileModifyDate  string `json:"FileModifyDate"`
	ExiftoolVersion string `json:"ExiftoolVersion"`
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

func computeDirectoryHash(path string) (string, error) {
	hash := sha1.New()

	err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if fileInfo.IsDir() {
			return nil
		}

		// Compute file hash and update the hash object
		fileHash, err := computeFileHash(filePath)
		if err != nil {
			return err
		}
		_, err = hash.Write([]byte(fileHash))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	hashBytes := hash.Sum(nil)
	hashValue := hex.EncodeToString(hashBytes)

	return hashValue, nil
}

func connectToMongoDB(dbType, host, port, dbUser, dbPwd, dbName, collectionName string) error {
	// Construct MongoDB connection URI
	mongodbURI := dbType + "://" + host + ":" + port

	// Configure the client connection
	clientOptions := options.Client().ApplyURI(mongodbURI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Check if the connection was successful
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Access the specified database and collection
	db := client.Database(dbName)
	fileCollection = db.Collection(collectionName)

	return nil
}

func insertFileDataIntoMongoDB(data FileData) error {
	_, err := fileCollection.InsertOne(context.Background(), data)
	if err != nil {
		return fmt.Errorf("failed to insert file data into MongoDB: %v", err)
	}

	return nil
}

func readDirectory(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory: %s", err)
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			dirHash, err := computeDirectoryHash(filePath)
			if err != nil {
				fmt.Printf("Error computing directory hash: %v\n", err)
				continue
			}

			dirData := FileData{
				IsDirectory:     true,
				FileSizeRaw:     0,  // Set file size to 0 for directories
				FileHash:        "", // Set file hash to empty string for directories
				SourceFileHash:  "", // Set source file hash to empty string for directories
				DirectoryHash:   dirHash,
				FilePermissions: getPermissions(file.Mode()), // Convert file permissions to string
				FileName:        file.Name(),
				SourceFile:      filePath,
				Directory:       path,
				FileModifyDate:  file.ModTime().Format(time.RFC3339),
				ExiftoolVersion: getExiftoolVersion(),
			}

			err = insertFileDataIntoMongoDB(dirData)
			if err != nil {
				fmt.Printf("Error inserting directory data into MongoDB: %v\n", err)
			}

			err = readDirectory(filePath)
			if err != nil {
				fmt.Printf("Error reading directory %s: %v\n", filePath, err)
			}
		} else {
			fileHash, err := computeFileHash(filePath)
			if err != nil {
				fmt.Printf("Error computing file hash: %v\n", err)
				continue
			}

			fileData := FileData{
				IsDirectory:     false,
				FileSizeRaw:     file.Size(),
				FileHash:        fileHash,
				SourceFileHash:  fileHash,                    // Set source file hash to file hash for files
				DirectoryHash:   "",                          // Set directory hash to empty string for files
				FilePermissions: getPermissions(file.Mode()), // Convert file permissions to string
				FileName:        file.Name(),
				SourceFile:      filePath,
				Directory:       filepath.Dir(filePath), // Update to save the parent directory instead of the root directory
				FileModifyDate:  file.ModTime().Format(time.RFC3339),
				ExiftoolVersion: getExiftoolVersion(),
			}

			// Extract and save Exiftool data for files
			exiftoolData, err := extractExiftoolData(filePath)
			if err != nil {
				fmt.Printf("Error extracting Exiftool data for file %s: %v\n", filePath, err)
			} else {
				fileData.ExiftoolVersion = exiftoolData
			}

			err = insertFileDataIntoMongoDB(fileData)
			if err != nil {
				fmt.Printf("Error inserting file data into MongoDB: %v\n", err)
			}
		}
	}

	return nil
}

func getExiftoolVersion() string {
	cmd := exec.Command("exiftool", "-ver")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error getting exiftool version: %v\n", err)
		return ""
	}

	return strings.TrimSpace(string(output))
}

// Function to extract Exiftool data for a file
func extractExiftoolData(filePath string) (string, error) {
	cmd := exec.Command("exiftool", "-j", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to extract Exiftool data: %v", err)
	}

	return string(output), nil
}

// Function to convert file permissions to r-w format
func getPermissions(perm os.FileMode) string {
	permStr := ""

	if perm&os.ModePerm == 0 {
		return "no access"
	}

	if perm&os.ModeDir != 0 {
		permStr += "d"
	} else {
		permStr += "-"
	}

	// Owner permissions
	if perm&0400 != 0 {
		permStr += "r"
	} else {
		permStr += "-"
	}

	if perm&0200 != 0 {
		permStr += "w"
	} else {
		permStr += "-"
	}

	if perm&0100 != 0 {
		permStr += "x"
	} else {
		permStr += "-"
	}

	// Group permissions
	if perm&0040 != 0 {
		permStr += "r"
	} else {
		permStr += "-"
	}

	if perm&0020 != 0 {
		permStr += "w"
	} else {
		permStr += "-"
	}

	if perm&0010 != 0 {
		permStr += "x"
	} else {
		permStr += "-"
	}

	// Other permissions
	if perm&0004 != 0 {
		permStr += "r"
	} else {
		permStr += "-"
	}

	if perm&0002 != 0 {
		permStr += "w"
	} else {
		permStr += "-"
	}

	if perm&0001 != 0 {
		permStr += "x"
	} else {
		permStr += "-"
	}

	return permStr
}

func main() {
	// MongoDB configuration
	dbType := "mongodb"
	dbHost := "localhost"
	dbPort := "27017"
	dbUser := "your-username"
	dbPwd := "your-password"
	dbName := "sopie"
	collectionName := "files"

	// Connect to MongoDB
	err := connectToMongoDB(dbType, dbHost, dbPort, dbUser, dbPwd, dbName, collectionName)
	if err != nil {
		fmt.Printf("Error connecting to MongoDB: %v\n", err)
		os.Exit(1)
	}

	// Capture and stringify path
	flag.Parse()
	basePath := *path

	// Read the directory
	err = readDirectory(basePath)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Directory and file data successfully inserted into MongoDB!")
}
