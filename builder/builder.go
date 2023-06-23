package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var root *string
var watcher *bool
var fileCollection *mongo.Collection

func init() {
	path = flag.String("path", "/home/delimp/Downloads/OPIe", "full path")
	root = flag.String("root", "", "root path")
	watcher = flag.Bool("watcher", false, "watcher")
}

func main() {
	flag.Parse()
	pathValue := *path
	rootValue := *root
	if rootValue == "" {
		rootValue = filepath.Dir(pathValue)
	}

	// Connect to MongoDB
	collection, err := connectToMongoDB("mongodb", "localhost", "27017", "localAdmin", "Army89Run!", "sopie", "files-test")
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		return
	}

	//	Process the path
	processPath(collection, pathValue, rootValue, watcher)
	// fmt.Println("Data saved to MongoDB successfully.")
}

// Process the path
func processPath(collection *mongo.Collection, pathValue, rootValue string, watcherValue *bool) {
	// Get file information
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		fmt.Printf("Failed to read file info: %v\n", err)
		return
	}

	if fileInfo.IsDir() {
		dirInfo := compileDirectoryData(pathValue, rootValue)
		// Save the directory data to MongoDB
		err = saveDataToDB(collection, dirInfo)
		if err != nil {
			fmt.Println("Failed to save data to MongoDB: ", err)
			return
		}

		// If it's a directory, process its contents
		// Open the directory
		dir, err := os.Open(pathValue)
		if err != nil {
			fmt.Println("Failed to open directory: ", err)
			return
		}
		defer dir.Close()

		// Read all the directory entries
		entries, err := dir.Readdir(-1)
		if err != nil {
			fmt.Println("Failed to read directory entries: ", err)
			return
		}

		// Loop over the directory entries and process each one
		for _, entry := range entries {
			entryPath := filepath.Join(pathValue, entry.Name())

			// Process files and directories recursively
			processPath(collection, entryPath, rootValue, watcherValue)
		}

	} else {
		// If the path is a file, process it
		fileData := compileFileData(pathValue, rootValue)

		if err != nil {
			fmt.Println("Failed to compile file data: ", err)
			return
		}
		// Save the file data to MongoDB
		err = saveDataToDB(collection, fileData)
		if err != nil {
			fmt.Println("Failed to save data to MongoDB: ", err)
			return
		}
	}
}

// Read a file's info using lstat
func readFileInfo(filePath string) (os.FileInfo, error) {
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	return fileInfo, nil
}

// Compile directory data
func compileDirectoryData(pathValue, rootValue string) map[string]string {
	// Get file information
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		fmt.Printf("Failed to read file info: %v\n", err)
		return nil
	}
	// Structure the file information
	sourceFile := pathValue
	directoryName := filepath.Dir(sourceFile)
	fileName := fileInfo.Name()
	size := fileInfo.Size()
	sizeString := strconv.FormatInt(size, 10)
	mode := fileInfo.Mode()
	modeString := mode.String()
	modTime := fileInfo.ModTime()
	modTimeString := modTime.Format("2006-01-02 15:04:05")
	sourcePathHash := computeStringHash(sourceFile)
	directoryHash := computeStringHash(directoryName)
	ancestorPaths := ancestryPaths(pathValue, rootValue)
	ancestorPathsString := strings.Join(ancestorPaths, ", ")
	ancestorPathHashes := ancestryPathHashes(ancestorPaths)
	uuid := sourcePathHash + ":" + directoryHash

	dirInfo := map[string]string{
		"_id":                uuid,
		"SourceFile":         sourceFile,
		"DirectoryName":      directoryName,
		"FileName":           fileName,
		"FileSizeRaw":        sizeString,
		"FileMode":           modeString,
		"FileModTime":        modTimeString,
		"SourcePathHash":     sourcePathHash,
		"DirectoryHash":      directoryHash,
		"AncestryPaths":      ancestorPathsString,
		"AncestryPathHashes": strings.Join(ancestorPathHashes, ", "),
		"IsDirectory":        "true",
	}
	return dirInfo
}

// Read a file's exif data and then flatten it using the flatten function
func readExifData(filePath string) (map[string]string, error) {
	cmd := exec.Command("exiftool", "-j", filePath)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	if err := json.Unmarshal(stdout, &data); err != nil {
		// Handle the case when the file has no EXIF data
		emptyData := make(map[string]string)
		return emptyData, nil
	}
	result := make(map[string]string)
	for k, v := range data[0] {
		flatten(result, k, reflect.ValueOf(v))
	}
	return result, nil
}

// Comple file data
func compileFileData(pathValue, rootValue string) map[string]string {
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		fmt.Printf("Failed to read file info: %v\n", err)
		return nil
	}
	// Read file's exif data using exiftool
	exifData, err := readExifData(pathValue)
	if err != nil {
		exifData, err = compileFileDataNoExif(pathValue, rootValue)
		if err != nil {
			fmt.Println("Failed to read exif data: ", err)
			return nil
		}
	}

	// Add file information not returned by exiftool
	sourcePathHash := computeStringHash(pathValue)
	directoryName := filepath.Dir(pathValue)
	directoryHash := computeStringHash(directoryName)
	fileHash := computeFileHash(pathValue)
	fsName := fileInfo.Name()
	fsExtension := filepath.Ext(fsName)
	size := fileInfo.Size()
	sizeString := strconv.FormatInt(size, 10)
	mode := fileInfo.Mode()
	modeString := mode.String()
	modTime := fileInfo.ModTime()
	modTimeString := modTime.Format("2006-01-02 15:04:05")
	ancestorPaths := ancestryPaths(pathValue, rootValue)
	ancestorPathsString := strings.Join(ancestorPaths, ", ")
	ancestorPathHashes := ancestryPathHashes(ancestorPaths)
	uuid := sourcePathHash + ":" + fileHash

	exifData["_id"] = uuid
	exifData["SourcePathHash"] = sourcePathHash
	exifData["DirectoryHash"] = directoryHash
	exifData["FileHash"] = fileHash
	exifData["FileSizeRaw"] = sizeString
	exifData["FileMode"] = modeString
	exifData["FileModTime"] = modTimeString
	exifData["AncestryPaths"] = ancestorPathsString
	exifData["AncestryPathHashes"] = strings.Join(ancestorPathHashes, ", ")
	exifData["FileTypeExtension"] = fsExtension
	exifData["IsDirectory"] = "false"

	return exifData
}

// Compile file data from files that return no exif data
func compileFileDataNoExif(pathValue, rootValue string) (map[string]string, error) {
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		fmt.Printf("Failed to read file info: %v\n", err)
		return nil, err
	}
	// Structure the file information
	sourceFile := pathValue
	directoryName := filepath.Dir(sourceFile)
	fileName := fileInfo.Name()
	size := fileInfo.Size()
	sizeString := strconv.FormatInt(size, 10)
	mode := fileInfo.Mode()
	modeString := mode.String()
	modTime := fileInfo.ModTime()
	modTimeString := modTime.Format("2006-01-02 15:04:05")
	sourcePathHash := computeStringHash(sourceFile)
	directoryHash := computeStringHash(directoryName)
	ancestorPaths := ancestryPaths(pathValue, rootValue)
	ancestorPathsString := strings.Join(ancestorPaths, ", ")
	ancestorPathHashes := ancestryPathHashes(ancestorPaths)
	fileHash := computeFileHash(pathValue)
	uuid := sourcePathHash + ":" + fileHash

	result := map[string]string{
		"_id":                uuid,
		"SourceFile":         sourceFile,
		"DirectoryName":      directoryName,
		"FileName":           fileName,
		"FileSizeRaw":        sizeString,
		"FileMode":           modeString,
		"FileModTime":        modTimeString,
		"IsDirectory":        "false",
		"SourcePathHash":     sourcePathHash,
		"DirectoryHash":      directoryHash,
		"AncestryPaths":      ancestorPathsString,
		"AncestryPathHashes": strings.Join(ancestorPathHashes, ", "),
		"FileHash":           fileHash,
	}
	return result, nil
}

// DATABASE FUNCTIONS
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

// UTILITY FUNCTIONS
// Compute the sha1 hash of a string
func computeStringHash(input string) string {
	hash := sha1.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)
	hashValue := hex.EncodeToString(hashBytes)
	return hashValue
}

// Identify the ancestry paths
func ancestryPaths(pathValue, rootValue string) []string {
	file := pathValue
	root := rootValue

	// Get the ancestry paths
	var paths []string
	for {
		file = filepath.Dir(file)
		paths = append(paths, file)
		if file == root {
			break
		}
	}
	return paths
}

// Compute ancestry path hashes
func ancestryPathHashes(ancestryPaths []string) []string {
	var hashes []string
	for _, path := range ancestryPaths {
		hash := computeStringHash(path)
		hashes = append(hashes, hash)
	}
	return hashes
}

// Compute the sha1 hash of a file
func computeFileHash(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// // Read Exifdata using exiftool and flatten the data returned into a map
// func readExifData(filePath string) (map[string]string, error) {
// 	cmd := exec.Command("exiftool", "-j", filePath)
// 	stdout, err := cmd.Output()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var data []map[string]interface{}
// 	if err := json.Unmarshal(stdout, &data); err != nil {
// 		// Handle the case when the file has no EXIF data
// 		emptyData := make(map[string]string)
// 		return emptyData, nil
// 	}
// 	result := make(map[string]string)
// 	for k, v := range data[0] {
// 		flatten(result, k, reflect.ValueOf(v))
// 	}
// 	return result, nil
// }

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
