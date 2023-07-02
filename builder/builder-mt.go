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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	DbType        string   `json:"DbType"`
	Host          string   `json:"Host"`
	Port          string   `json:"Port"`
	DbUser        string   `json:"DbUser"`
	DbPwd         string   `json:"DbPwd"`
	DbName        string   `json:"DbName"`
	FileColl      string   `json:"FileColl"`
	TreeColl      string   `json:"TreeColl"`
	MaxGoroutines int      `json:"maxGoroutines"`
	NoExif        []string `json:"NoExif"`
	Watcher       []string `json:"Watcher"`
	Root          string   `json:"root"`
	Path          string   `json:"path"`
}

var config *string
var path *string
var root *string
var watcher *bool
var fileCollection *mongo.Collection
var workerCount = 25
var workerPool = make(chan struct{}, workerCount)
var counter int

func init() {
	// Read the configuration file
	config, err := readConfig("conf.json")
	if err != nil {
		fmt.Printf("Failed to read configuration file: %v\n", err)
		return
	}

	path = flag.String("path", config.Path, "full path")
	root = flag.String("root", config.Root, "root path")
	watcher = flag.Bool("watcher", false, "watcher")
}

func main() {
	flag.Parse()

	startTime := time.Now()
	config, err := readConfig("conf.json")
	if err != nil {
		log.Fatalf("Failed to read configuration file: %v", err)
	}

	pathValue := *path
	rootValue := *root
	// If root is not passed, we must assume that the path is the root
	if rootValue == "" {
		rootValue = pathValue
	}

	if config.MaxGoroutines > 0 {
		workerCount = config.MaxGoroutines
		workerPool = make(chan struct{}, workerCount)
	}

	collection, err := connectToMongoDB(config.DbType, config.Host, config.Port, config.DbUser, config.DbPwd, config.DbName, config.FileColl)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go processPath(collection, *path, *root, *watcher, wg)
	wg.Wait() // Wait for all goroutines to finish.
	elapsedTime := time.Since(startTime)
	log.Printf("Execution time: %s", elapsedTime)
}

func readConfig(filename string) (Config, error) {
	var config Config

	// Read the JSON file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read configuration file: %v", err)
	}

	// Unmarshal the JSON data into the Config struct
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal configuration data: %v", err)
	}

	return config, nil
}

// Process the path
func processPath(collection *mongo.Collection, pathValue, rootValue string, watcherValue bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get file information once
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		log.Printf("Failed to read file info: %v\n", err)
		return
	}

	// Create a channel for completion signals
	complete := make(chan bool)

	// Submit task to the worker pool
	workerPool <- struct{}{}
	go func() {
		err := runCompileAndWrite(collection, pathValue, rootValue, watcherValue, fileInfo)
		<-workerPool // Release the worker slot when completed
		if err != nil {
			log.Printf("Error processing path: %v\n", err)
		}
		complete <- true
	}()

	if fileInfo.IsDir() && !isSymbolicLink(fileInfo) {
		// If it's a directory and not a symbolic link, process its contents
		// Open the directory
		dir, err := os.Open(pathValue)
		if err != nil {
			log.Printf("Failed to open directory: %v\n", err)
			return
		}
		defer dir.Close()

		// Read all the directory entries
		entries, err := dir.Readdir(-1)
		if err != nil {
			log.Printf("Failed to read directory entries: %v\n", err)
			return
		}

		// Loop over the directory entries and process each one
		for _, entry := range entries {
			entryPath := filepath.Join(pathValue, entry.Name())
			wg.Add(1)
			go processPath(collection, entryPath, rootValue, watcherValue, wg)
		}
	}

	// Wait for the completion signal
	<-complete
}

// Determine file type and do both compileData and saveDataToDB
func runCompileAndWrite(collection *mongo.Collection, pathValue, rootValue string, watcherValue bool, fileInfo os.FileInfo) error {
	goroutineNumber := incrementCounter()
	fmt.Printf("Goroutine %d started\n", goroutineNumber)
	dataInfo, err := compileData(pathValue, rootValue, fileInfo)
	if err != nil {
		return err
	}

	err = saveDataToDB(collection, dataInfo)
	if err != nil {
		return fmt.Errorf("failed to save data to MongoDB: %v", err)
	}

	return nil
}

// Compile directory or file data
func compileData(pathValue, rootValue string, fileInfo os.FileInfo) (map[string]string, error) {
	if isSymbolicLink(fileInfo) {
		// For symlinks, handle symlink data
		linkPath, err := os.Readlink(pathValue)
		if err != nil {
			return nil, err
		}
		symlinkIsDir := ""
		if fileInfo.IsDir() {
			symlinkIsDir = "true"
		} else {
			symlinkIsDir = "false"
		}

		symlinkInfo := map[string]string{
			"_id":                computeStringHash(pathValue),
			"SourceFile":         pathValue,
			"DirectoryName":      filepath.Dir(pathValue),
			"FileName":           fileInfo.Name(),
			"FileSizeRaw":        strconv.FormatInt(fileInfo.Size(), 10),
			"FileMode":           fileInfo.Mode().String(),
			"FileModTime":        fileInfo.ModTime().Format("2006-01-02 15:04:05"),
			"SourcePathHash":     computeStringHash(pathValue),
			"DirectoryHash":      computeStringHash(filepath.Dir(pathValue)),
			"AncestryPaths":      strings.Join(ancestryPaths(pathValue, rootValue), ", "),
			"AncestryPathHashes": strings.Join(ancestryPathHashes(ancestryPaths(pathValue, rootValue)), ", "),
			"IsDirectory":        symlinkIsDir,
			"IsSymLink":          "true",
			"SymlinkDestination": linkPath,
		}
		return symlinkInfo, nil
	} else if fileInfo.IsDir() {
		// For directories, compile directory data
		dirInfo := map[string]string{
			"_id":                computeStringHash(pathValue),
			"SourceFile":         pathValue,
			"DirectoryName":      filepath.Dir(pathValue),
			"FileName":           fileInfo.Name(),
			"FileSizeRaw":        strconv.FormatInt(fileInfo.Size(), 10),
			"FileMode":           fileInfo.Mode().String(),
			"FileModTime":        fileInfo.ModTime().Format("2006-01-02 15:04:05"),
			"SourcePathHash":     computeStringHash(pathValue),
			"DirectoryHash":      computeStringHash(filepath.Dir(pathValue)),
			"AncestryPaths":      strings.Join(ancestryPaths(pathValue, rootValue), ", "),
			"AncestryPathHashes": strings.Join(ancestryPathHashes(ancestryPaths(pathValue, rootValue)), ", "),
			"IsDirectory":        "true",
		}
		return dirInfo, nil
	} else {
		// For files, compile file data
		exifData, err := readExifData(pathValue)
		if err != nil {
			// If exif data is not available, compile data without exif
			fileInfo := map[string]string{
				"_id":                computeStringHash(pathValue),
				"SourceFile":         pathValue,
				"DirectoryName":      filepath.Dir(pathValue),
				"FileName":           fileInfo.Name(),
				"FileSizeRaw":        strconv.FormatInt(fileInfo.Size(), 10),
				"FileMode":           fileInfo.Mode().String(),
				"FileModTime":        fileInfo.ModTime().Format("2006-01-02 15:04:05"),
				"IsDirectory":        "false",
				"SourcePathHash":     computeStringHash(pathValue),
				"DirectoryHash":      computeStringHash(filepath.Dir(pathValue)),
				"AncestryPaths":      strings.Join(ancestryPaths(pathValue, rootValue), ", "),
				"AncestryPathHashes": strings.Join(ancestryPathHashes(ancestryPaths(pathValue, rootValue)), ", "),
				"FileHash":           computeFileHash(pathValue),
			}
			return fileInfo, nil
		}

		// Add additional file information to exif data
		exifData["_id"] = computeStringHash(pathValue)
		exifData["SourcePathHash"] = computeStringHash(pathValue)
		exifData["DirectoryHash"] = computeStringHash(filepath.Dir(pathValue))
		exifData["FileHash"] = computeFileHash(pathValue)
		exifData["FileSizeRaw"] = strconv.FormatInt(fileInfo.Size(), 10)
		exifData["FileMode"] = fileInfo.Mode().String()
		exifData["FileModTime"] = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		exifData["AncestryPaths"] = strings.Join(ancestryPaths(pathValue, rootValue), ", ")
		exifData["AncestryPathHashes"] = strings.Join(ancestryPathHashes(ancestryPaths(pathValue, rootValue)), ", ")
		exifData["FileTypeExtension"] = filepath.Ext(fileInfo.Name())
		exifData["IsDirectory"] = "false"

		return exifData, nil
	}
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

// Read a file's info using lstat
func readFileInfo(filePath string) (os.FileInfo, error) {
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	return fileInfo, nil
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

// Function to check if a file info represents a symbolic link
func isSymbolicLink(fileInfo os.FileInfo) bool {
	return fileInfo.Mode()&os.ModeSymlink != 0
}

// Flatten nested map data
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

// Flatten nested map data with map keys
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

// Flatten nested map data with slice values
func flattenSlice(result map[string]string, prefix string, v reflect.Value) {
	prefix = prefix + "."
	for i := 0; i < v.Len(); i++ {
		flatten(result, fmt.Sprintf("%s%d", prefix, i), v.Index(i))
	}
}

func incrementCounter() int {
	counter++
	return counter
}
