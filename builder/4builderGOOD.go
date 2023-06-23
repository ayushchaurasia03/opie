package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var fileCollection *mongo.Collection

func init() {
	path = flag.String("path", "/home/delimp/Downloads/OPIe", "full path")
}

// Compute the sha1 hash of a file
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

// Read a file's info using lstat
func readFileInfo(filePath string) (os.FileInfo, error) {
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// Compute the sha1 hash of a string
func computeStringHash(input string) string {
	hash := sha1.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)
	hashValue := hex.EncodeToString(hashBytes)
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	return hashValue
}

// Compute an array of ancestor paths
func ancestorPaths(file, root string) []string {
	var paths []string
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	for {
		file = filepath.Dir(file)
		paths = append(paths, file)
		if file == root {
			break
		}
	}
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	return paths
}

// Compute an array of ancestor hashes
func ancestorHashes(strs []string) []string {
	var hashes []string
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	for _, str := range strs {
		hash := sha1.New()
		hash.Write([]byte(str))
		sha1Hash := fmt.Sprintf("%x", hash.Sum(nil))
<<<<<<< HEAD
		hashes = append(hashes, sha1Hash)
	}
=======

		hashes = append(hashes, sha1Hash)
	}

>>>>>>> Refactor-GCH
	return hashes
}

// Read a file's exif data and then flatten it using the flatten function
func readExifData(filePath string) (map[string]string, error) {
	cmd := exec.Command("exiftool", "-j", filePath)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	var data []map[string]interface{}
	if err := json.Unmarshal(stdout, &data); err != nil {
		// Handle the case when the file has no EXIF data
		emptyData := make(map[string]string)
		return emptyData, nil
	}
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	result := make(map[string]string)
	for k, v := range data[0] {
		flatten(result, k, reflect.ValueOf(v))
	}
<<<<<<< HEAD
=======

>>>>>>> Refactor-GCH
	return result, nil
}

// Extracted from flatjson.go
func flatten(result map[string]string, prefix string, v reflect.Value) {
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			result[prefix] = "true"
		} else {
			result[prefix] = "false"
		}
	case reflect.Int:
		result[prefix] = fmt.Sprintf("%d", v.Int())
	case reflect.Float64:
		result[prefix] = fmt.Sprintf("%f", v.Float())
	case reflect.Map:
		flattenMap(result, prefix, v)
	case reflect.Slice:
		flattenSlice(result, prefix, v)
	case reflect.String:
		result[prefix] = v.String()
	default:
		panic(fmt.Sprintf("Couldn't deal with: %s", v))
	}
}

func flattenMap(result map[string]string, prefix string, v reflect.Value) {
	for _, k := range v.MapKeys() {
		if k.Kind() == reflect.Interface {
			k = k.Elem()
		}

		if k.Kind() != reflect.String {
			panic(fmt.Sprintf("%s: map key is not string: %s", prefix, k))
		}

		flatten(result, fmt.Sprintf("%s.%s", prefix, k.String()), v.MapIndex(k))
	}
}

func flattenSlice(result map[string]string, prefix string, v reflect.Value) {
	prefix = prefix + "."
	for i := 0; i < v.Len(); i++ {
		flatten(result, fmt.Sprintf("%s%d", prefix, i), v.Index(i))
	}
}

// Read a file's data when it has no EXIF information
func readNoExifData(filePath string) (map[string]string, error) {
	fileInfo, err := readFileInfo(filePath)
	if err != nil {
		return nil, err
	}

	size := fileInfo.Size()
	sizeStr := strconv.FormatInt(size, 10)

	mode := fileInfo.Mode()
	modeStr := mode.String()

	modTime := fileInfo.ModTime()
	modTimeStr := modTime.Format("2006-01-02 15:04:05")

	sourcePathHash := computeStringHash(filePath)
	directoryPathValue := filepath.Dir(filePath)
	directoryHash := computeStringHash(directoryPathValue)

	// Get Ancestors paths and hashes
	ancestors := ancestorPaths(filePath, *path)
	ancestorHashes := ancestorHashes(ancestors)

	result := map[string]string{
		"_id":            sourcePathHash + ":" + directoryHash,
		"SourceFile":     filePath,
		"Directory":      directoryPathValue,
		"FileName":       fileInfo.Name(),
		"FsSizeRaw":      sizeStr,
		"FileMode":       modeStr,
		"FileModTime":    modTimeStr,
		"IsDirectory":    "false",
		"SourcePathHash": sourcePathHash,
		"DirectoryHash":  directoryHash,
		"AncestorPaths":  fmt.Sprintf("%v", ancestors),
		"AncestorHashes": fmt.Sprintf("%v", ancestorHashes),
	}

	return result, nil
}

// A function to compile file data
func compileFileData(filePath string) (map[string]string, error) {
	fileInfo, err := readFileInfo(filePath)
	if err != nil {
		return nil, err
	}

	mode := fileInfo.Mode()
	modeStr := mode.String()

	modTime := fileInfo.ModTime()
	modTimeStr := modTime.Format("2006-01-02 15:04:05")

	// Compute the sha1 hash of the file by calling computeFileHash
	fileHash, err := computeFileHash(filePath)
	if err != nil {
		return nil, err
	}

	// Compute the sha1 hash of the file source path by calling computeStringHash
	sourcePathHash := computeStringHash(filePath)

	// Compute the sha1 hash of the file's directory by calling computeStringHash
	directoryPathValue := filepath.Dir(filePath)
	directoryHash := computeStringHash(directoryPathValue)

	// Set UUID - for files, this is the source path hash + file hash
	UUID := sourcePathHash + ":" + fileHash

	// Read the file's exif data by calling readExifData
	exifData, err := readExifData(filePath)
	if err != nil {
		// Handle the case when the file has no EXIF data
		exifData, err = readNoExifData(filePath)
		if err != nil {
			return nil, err
		}
	}

	exifData["_id"] = UUID
	exifData["sourcePathHash"] = sourcePathHash
	exifData["directoryHash"] = directoryHash
	exifData["fileHash"] = fileHash
	exifData["fsName"] = fileInfo.Name()
	exifData["fsSizeRaw"] = strconv.FormatInt(fileInfo.Size(), 10)
	exifData["fsMode"] = modeStr
	exifData["fsModTime"] = modTimeStr
	exifData["isDirectory"] = "false"

	return exifData, nil
}

// Connect to MongoDB and return the collection
func connectToMongoDB(dbType, host, port, dbUser, dbPwd, dbName, collectionName string) (*mongo.Collection, error) {
	// Construct MongoDB connection URI
	mongodbURI := dbType + "://" + dbUser + ":" + dbPwd + "@" + host + ":" + port

	// Configure the client connection
	clientOptions := options.Client().ApplyURI(mongodbURI)

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Check if the connection was successful
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Access the specified database and collection
	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	return collection, nil
}

// Save data to MongoDB
func saveDataToDB(collection *mongo.Collection, data map[string]string) error {
	// Convert the data map to BSON
	doc := bson.M{}
	for k, v := range data {
		doc[k] = v
	}

	// Set the filter to check if the document with the given _id already exists
	filter := bson.M{"_id": doc["_id"]}

	// Set the update to replace the existing document with the new data
	update := bson.M{"$set": doc}

	// Set the options for upsert (create if not exists)
	options := options.Update().SetUpsert(true)

	// Perform the upsert operation in MongoDB
	_, err := collection.UpdateOne(context.Background(), filter, update, options)
	if err != nil {
		return err
	}

	return nil
}

func logError(err error, filePath string) {
	fmt.Printf("Error occurred for file: %s\n%s\n", filePath, err.Error())
}

// A function to compile directory data
func compileDirectoryData(pathValue, rootPath string) map[string]string {
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		logError(err, pathValue)
		return nil
	}

	size := fileInfo.Size()
	sizeStr := strconv.FormatInt(size, 10)

	mode := fileInfo.Mode()
	modeStr := mode.String()

	modTime := fileInfo.ModTime()
	modTimeStr := modTime.Format("2006-01-02 15:04:05")

	sourcePathHash := computeStringHash(pathValue)
	directoryPathValue := filepath.Dir(pathValue)
	directoryHash := computeStringHash(directoryPathValue)

	uuid := sourcePathHash + ":" + directoryHash

	dirInfo := map[string]string{
		"_id":            uuid,
		"SourceFile":     pathValue,
		"Directory":      directoryPathValue,
		"FileName":       fileInfo.Name(),
		"FsSizeRaw":      sizeStr,
		"FileMode":       modeStr,
		"FileModTime":    modTimeStr,
		"IsDirectory":    "true",
		"SourcePathHash": sourcePathHash,
		"DirectoryHash":  directoryHash,
	}

	return dirInfo
}

// Process a path (directory or file)
func processPath(collection *mongo.Collection, pathValue, rootPath string) {
	// Get file information
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		logError(err, pathValue)
		return
	}

	// Save directory data
	if fileInfo.IsDir() {
		dirInfo := compileDirectoryData(pathValue, rootPath)

		// Save the directory data to MongoDB
		err = saveDataToDB(collection, dirInfo)
		if err != nil {
			logError(err, pathValue)
		}

		// If it's a directory, process its contents
		// Open the directory
		dir, err := os.Open(pathValue)
		if err != nil {
			logError(err, pathValue)
			return
		}
		defer dir.Close()

		// Read all the directory entries
		entries, err := dir.Readdir(-1)
		if err != nil {
			logError(err, pathValue)
			return
		}

		// Loop over the directory entries and process each one
		for _, entry := range entries {
			entryPath := filepath.Join(pathValue, entry.Name())

			// Process files and directories recursively
			processPath(collection, entryPath, rootPath)
		}
	} else {
		// If the path is a file, process the file
		fileData, err := compileFileData(pathValue)
		if err != nil {
			logError(err, pathValue)
			return
		}

		// Save the file data to MongoDB
		err = saveDataToDB(collection, fileData)
		if err != nil {
			logError(err, pathValue)
		}
	}
}

func main() {
	flag.Parse()
	pathValue := *path

	// Connect to MongoDB
<<<<<<< HEAD
	collection, err := connectToMongoDB("mongodb", "localhost", "27017", "localAdmin", "Army89Run!", "sopie", "testing")
=======
	collection, err := connectToMongoDB("mongodb", "localhost", "27017", "admin", "password", "sopie", "testing")
>>>>>>> Refactor-GCH
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		return
	}

	// Process the path
	processPath(collection, pathValue, pathValue)

	fmt.Println("Data saved to MongoDB successfully.")
}
