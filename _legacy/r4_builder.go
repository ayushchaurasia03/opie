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
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var path *string
var fileCollection *mongo.Collection

func init() {
	path = flag.String("path", "/home/delimp/Documents/test/", "full path")
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

	return hashValue
}

// Compute an array of ancestor paths
func ancestorPaths(pathValue, rootPath string) []string {
	var paths []string
	for {
		pathValue = filepath.Dir(pathValue)
		paths = append(paths, pathValue)
		if pathValue == rootPath {
			break
		}
	}
	return paths
}

// Compute an array of ancestor hashes
func ancestorHashes(jsonPaths []string) []string {
	var hashes []string
	for _, jsonPath := range jsonPaths {
		hash := sha1.New()
		hash.Write([]byte(jsonPath))
		sha1Hash := fmt.Sprintf("%x", hash.Sum(nil))

		hashes = append(hashes, sha1Hash)
	}
	return hashes
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

// Read a file's exif data and then flatten it using the flatten function
func readExifData(filePath string) (map[string]string, error) {
	cmd := exec.Command("exiftool", "-j", filePath)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(stdout, &data); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range data[0] {
		flatten(result, k, reflect.ValueOf(v))
	}

	return result, nil
}

// A function to compile directory data
func compileDirectoryData(pathValue string) string {
	fmt.Println(pathValue)
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		panic(err)
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

	nowTime := time.Now()
	timeNow := nowTime.Format("2006-01-02 15:04:05")
	timeHash := computeStringHash(timeNow)

	ancestors := ancestorPaths(pathValue, "/Users/greghacke/Pictures")
	ancestorHashes := ancestorHashes(ancestors)

	uuid := sourcePathHash + ":" + timeHash

	dirInfo := map[string]interface{}{
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
		"UpdateTime":     timeNow,
		"Ancestors":      strings.Join(ancestors, ", "),
		"AncestorHashes": strings.Join(ancestorHashes, ", "),
	}

	fileJson, errMar := json.MarshalIndent(dirInfo, "", "  ")
	if errMar != nil {
		fmt.Printf("err - %v\n", errMar)
	}
	var response = string(fileJson)
	return response
}

// Read a file's info using function readFileInfo
func compileFileData(pathValue string) string {
	fmt.Println(pathValue)
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		panic(err)
	}

	mode := fileInfo.Mode()
	modeStr := mode.String()

	modTime := fileInfo.ModTime()
	modTimeStr := modTime.Format("2006-01-02 15:04:05")

	// Compute the sha1 hash of the file by calling computeFileHash
	fileHash, err := computeFileHash(pathValue)
	if err != nil {
		panic(err)
	}

	// Compute the sha1 hash of the file source path by calling computeStringHash
	sourcePathHash := computeStringHash(pathValue)

	// Compute the sha1 hash of the file's directory by calling computeStringHash
	directoryPathValue := filepath.Dir(pathValue)
	directoryHash := computeStringHash(directoryPathValue)

	// Get Ancestors paths and hashes
	ancestors := ancestorPaths(pathValue, "/Users/greghacke/Pictures")
	ancestorHashes := ancestorHashes(ancestors)

	// Set UIID - for files this is the source path hash + file hash
	UUID := sourcePathHash + ":" + fileHash

	// Read the file's exif data by calling readExifData
	exifData, err := readExifData(pathValue)
	if err != nil {
		// If exif data cannot be read, treat it as a directory
		response := compileFileDataNoEXIF(pathValue)

		// Save the limited directory data to MongoDB
		err = saveDataToDB(response)
		if err != nil {
			panic(err)
		}

		return response
	}

	exifData["_id"] = UUID
	exifData["SourcePathHash"] = sourcePathHash
	exifData["DirectoryHash"] = directoryHash
	exifData["FileHash"] = fileHash
	exifData["FsName"] = fileInfo.Name()
	exifData["FsSizeRaw"] = strconv.FormatInt(fileInfo.Size(), 10)
	exifData["FsMode"] = modeStr
	exifData["FsModTime"] = modTimeStr
	exifData["IsDirectory"] = "false"
	exifData["Ancestors"] = strings.Join(ancestors, ", ")
	exifData["AncestorHashes"] = strings.Join(ancestorHashes, ", ")

	// Marshal outbound json
	fileJson, errUnm := json.MarshalIndent(&exifData, "", "  ")
	if errUnm != nil {
		fmt.Printf("err - %v\n", errUnm)
	}
	var response = string(fileJson)
	return response
}

// A function to compile file data for files that cannot be read with exiftool
func compileFileDataNoEXIF(pathValue string) string {
	fmt.Println(pathValue)
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		panic(err)
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

	// Get Ancestors paths and hashes
	ancestors := ancestorPaths(pathValue, "/Users/greghacke/Pictures")
	ancestorHashes := ancestorHashes(ancestors)

	uuid := sourcePathHash + ":" + directoryHash

	fileNoEXIFInfo := map[string]interface{}{
		"_id":            uuid,
		"SourceFile":     pathValue,
		"Directory":      directoryPathValue,
		"FileName":       fileInfo.Name(),
		"FsSizeRaw":      sizeStr,
		"FileMode":       modeStr,
		"FileModTime":    modTimeStr,
		"IsDirectory":    "false",
		"SourcePathHash": sourcePathHash,
		"DirectoryHash":  directoryHash,
		"Ancestors":      strings.Join(ancestors, ", "),
		"AncestorHashes": strings.Join(ancestorHashes, ", "),
	}

	fileJson, errMar := json.MarshalIndent(fileNoEXIFInfo, "", "  ")
	if errMar != nil {
		fmt.Printf("err - %v\n", errMar)
	}
	var response = string(fileJson)
	return response
}

// A function to connect to MongoDB
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

// Function to save data to MongoDB
func saveDataToDB(data string) error {
	// Unmarshal the JSON data into a map
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return err
	}

	// Insert the data into the MongoDB collection
	_, err := fileCollection.InsertOne(context.Background(), jsonData)
	if err != nil {
		return err
	}

	return nil
}

// A function to process a path (directory or file)
func processPath(pathValue string) error {
	// Get file information
	fileInfo, err := readFileInfo(pathValue)
	if err != nil {
		return err
	}

	// Save directory data
	if fileInfo.IsDir() {
		response := compileDirectoryData(pathValue)

		// Save the directory data to MongoDB
		err = saveDataToDB(response)
		if err != nil {
			return err
		}

		// If it's a directory, process its contents
		// Open the directory
		dir, err := os.Open(pathValue)
		if err != nil {
			return err
		}
		defer dir.Close()

		// Read all the directory entries
		entries, err := dir.Readdir(-1)
		if err != nil {
			return err
		}

		// Loop over the directory entries and process each one
		for _, entry := range entries {
			entryPath := filepath.Join(pathValue, entry.Name())

			// Process files and directories recursively
			err = processPath(entryPath)
			if err != nil {
				return err
			}
		}
	} else {
		// If the path is a file, process the file
		response := compileFileData(pathValue)

		// Save the file data to MongoDB
		err = saveDataToDB(response)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()
	pathValue := *path

	// Connect to MongoDB
	err := connectToMongoDB("mongodb", "localhost", "27017", "localAdmin", "Army89Run!", "sopie", "builder-test")
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		return
	}

	// Process the path
	err = processPath(pathValue)
	if err != nil {
		fmt.Printf("Failed to process path: %v\n", err)
		return
	}

	fmt.Println("Data saved to MongoDB successfully.")
}
