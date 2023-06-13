package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"path/filepath"
)

var (
	file *string
	root *string
)

func init() {
	file = flag.String("file", "/pat/to/your/file/name.txt", "full file path")
	root = flag.String("root", "/pat/to/your/file/", "root path")
}

func main() {
	flag.Parse()
	fileValue := *file
	rootValue := *root

	paths := getPathsBetween(fileValue, rootValue)
	hashes := createHashes(paths)
	fmt.Println(paths)
	fmt.Println(hashes)
}

func getPathsBetween(file, root string) []string {
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

func createHashes(strs []string) map[string]string {
	hashes := make(map[string]string)

	for _, str := range strs {
		hash := sha1.New()
		hash.Write([]byte(str))
		sha1Hash := fmt.Sprintf("%x", hash.Sum(nil))

		hashes[str] = sha1Hash
	}

	return hashes
}
