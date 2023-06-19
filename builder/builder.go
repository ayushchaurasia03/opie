package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var fileCollection *mongo.Collection

type FileData struct {
	ID              string                 `json:"_id,omitempty"`
	IsDirectory     bool                   `json:"IsDirectory"`
	FileSizeRaw     int64                  `json:"FileSizeRaw"`
	FileHash        string                 `json:"FileHash"`
	SourceFileHash  string                 `json:"SourceFileHash"`
	DirectoryHash   string                 `json:"DirectoryHash"`
	FilePermissions string                 `json:"FilePermissions"`
	FileName        string                 `json:"FileName"`
	SourceFile      string                 `json:"SourceFile"`
	Directory       string                 `json:"Directory"`
	FileModifyDate  string                 `json:"FileModifyDate"`
	ExiftoolVersion string                 `json:"ExiftoolVersion"`
	ExiftoolData    map[string]interface{} `json:"ExiftoolData"` // New field for Exiftool data
}

func init() {
	path = flag.String("path", "/home/delimp/Documents/test", "full path")
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

func computeDirectoryHash(path string, parentDirHash string) (string, error) {
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

		// Include file path, name, and hash in the directory hash computation
		_, err = hash.Write([]byte(filePath))
		if err != nil {
			return err
		}
		_, err = hash.Write([]byte(fileInfo.Name()))
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
	mongodbURI := dbType + "://" + dbUser + ":" + dbPwd + "@" + host + ":" + port

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
	data.ID = data.SourceFileHash + ":" + data.FileHash // Set the ID field as concatenation of SourceFileHash and FileHash

	// Create a new document without the _id field
	doc := bson.M{
		"is_directory":     data.IsDirectory,
		"file_size_raw":    data.FileSizeRaw,
		"file_hash":        data.FileHash,
		"source_file_hash": data.SourceFileHash,
		"directory_hash":   data.DirectoryHash,
		"file_permissions": data.FilePermissions,
		"file_name":        data.FileName,
		"source_file":      data.SourceFile,
		"directory":        data.Directory,
		"file_modify_date": data.FileModifyDate,
		"exiftool_version": data.ExiftoolVersion,
		"exiftool_data":    data.ExiftoolData,
		"_id":              data.ID,
	}

	_, err := fileCollection.InsertOne(context.Background(), doc)
	if err != nil {
		return fmt.Errorf("failed to insert file data into MongoDB: %v", err)
	}

	return nil
}

func readDirectory(path string, parentDirHash string) error {
	files, err := ioutil.ReadDir(path)

	if err != nil {
		return fmt.Errorf("error reading directory: %s", err)
	}

	dirHash, err := computeDirectoryHash(path, parentDirHash) // Compute directory hash based on the path and parent directory hash
	if err != nil {
		return fmt.Errorf("error computing directory hash: %s", err)
	}

	dirData := FileData{
		IsDirectory:     true,
		FileSizeRaw:     0,       // Set file size to 0 for directories
		FileHash:        dirHash, // Set directory hash as the file hash for directories
		SourceFileHash:  dirHash, // Set directory hash as the source file hash for directories
		DirectoryHash:   dirHash,
		FilePermissions: getPermissions(os.ModeDir), // Convert directory permissions to string
		FileName:        filepath.Base(path),
		SourceFile:      path,
		Directory:       filepath.Dir(path),              // Update to save the parent directory instead of the root directory
		FileModifyDate:  time.Now().Format(time.RFC3339), // Update to use current time
		ExiftoolVersion: getExiftoolVersion(),
	}

	err = insertFileDataIntoMongoDB(dirData)
	if err != nil {
		fmt.Printf("Error inserting directory data into MongoDB: %v\n", err)
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			err = readDirectory(filePath, dirHash) // Pass the directory hash as the parent directory hash
			if err != nil {
				fmt.Printf("Error reading subdirectory %s: %v\n", filePath, err)
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
				SourceFileHash:  dirHash,                     // Set directory hash as the source file hash for files
				DirectoryHash:   dirHash,                     // Set directory hash to the current directory's hash for files
				FilePermissions: getPermissions(file.Mode()), // Convert file permissions to string
				FileName:        file.Name(),
				SourceFile:      filePath,
				Directory:       path,
				FileModifyDate:  file.ModTime().Format(time.RFC3339),
				ExiftoolVersion: getExiftoolVersion(),
			}

			// Extract and save Exiftool data for files
			exiftoolData, err := extractExiftoolData(filePath)
			if err != nil {
				fmt.Printf("Error extracting Exiftool data for file %s: %v\n", filePath, err)
			} else {
				fileData.ExiftoolData = exiftoolData // Assign the extracted Exiftool data to the ExiftoolData field
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
func extractExiftoolData(filePath string) (map[string]interface{}, error) {
	cmd := exec.Command("exiftool", "-j", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract Exiftool data: %v", err)
	}

	var data []map[string]interface{}
	err = json.Unmarshal(output, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Exiftool data: %v", err)
	}

	if len(data) > 0 {
		return data[0], nil // Return the first object in the array
	}

	return nil, nil
}

// Function to convert file permissions to r-w format
func getPermissions(perm os.FileMode) string {
	permStr := ""

	// File type
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

	// Group permissions
	if perm&040 != 0 {
		permStr += "r"
	} else {
		permStr += "-"
	}

	if perm&020 != 0 {
		permStr += "w"
	} else {
		permStr += "-"
	}

	// Other permissions
	if perm&04 != 0 {
		permStr += "r"
	} else {
		permStr += "-"
	}

	if perm&02 != 0 {
		permStr += "w"
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
	dbUser := "localAdmin"
	dbPwd := "Army89Run!"
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
	err = readDirectory(basePath, "")
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Directory and file data successfully inserted into MongoDB!")
}
