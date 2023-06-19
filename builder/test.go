package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
)

var path *string
var fileCollection *mongo.Collection

func init() {
	path = flag.String("path", "/home/delimp/Documents/test", "full path")
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

// A function to compile directory data
func compileDirectoryData(pathValue string) string {
	// Read a file's info using function readFileInfo
	fileInfo, err := readFileInfo(*path)
	if err != nil {
		panic(err)
	}

	size := fileInfo.Size()
	sizeStr := strconv.FormatInt(size, 10)

	mode := fileInfo.Mode()
	modeStr := mode.String()

	modTime := fileInfo.ModTime()
	modTimeStr := modTime.Format("2006-01-02 15:04:05")

	// Compute the sha1 hash of the file source path by calling computeStringHash
	sourcePathHash := computeStringHash(*path)

	// Compute the sha1 hash of the file's directory by calling computeStringHash
	directoryPathValue := filepath.Dir(pathValue)
	directoryHash := computeStringHash(directoryPathValue)

	// Set UIID - for directory, this is the source path hash + directory hash
	UUID := sourcePathHash + ":" + directoryHash
	// fmt.Println("_id:", UUID)

	//exifData["_id"] = UUID
	//exifData["sourcePathHash"] = sourcePathHash
	//exifData["directoryHash"] = directoryHash
	//exifData["fileHash"] = fileHash
	//exifData["fsName"] = fileInfo.Name()
	//exifData["fsSizeRaw"] = sizeStr
	//exifData["fsMode"] = modeStr
	//exifData["fsModTime"] = modTimeStr
	//exifData["isDirectory"] = "false"

	fmt.Println("_id:", UUID)
	fmt.Println("Source File:", pathValue)
	fmt.Println("File Name:", fileInfo.Name())
	fmt.Println("File Size:", sizeStr)
	fmt.Println("File Mode:", modeStr)
	fmt.Println("File Mod Time:", modTimeStr)
	fmt.Println("isDirectory:", "true")
	fmt.Println("sourcePathHash:", sourcePathHash)
	fmt.Println("directoryHash:", directoryHash)

	return ""
}

func main() {
	flag.Parse()
	pathValue := *path

	response := compileDirectoryData(pathValue)
	fmt.Println(response)

}
