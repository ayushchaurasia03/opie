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
	file = flag.String("file", "/path/to/your/file/name.txt", "full file path")
	root = flag.String("root", "/path", "root path")
}

func main() {
	flag.Parse()
	fileValue := *file
	rootValue := *root

	paths := ancestorPaths(fileValue, rootValue)
	hashes := ancestorHashes(paths)
	fmt.Println(paths)
	fmt.Println(hashes)
}

func ancestorPaths(file, root string) []string {
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

func ancestorHashes(strs []string) []string {
	var hashes []string

	for _, str := range strs {
		hash := sha1.New()
		hash.Write([]byte(str))
		sha1Hash := fmt.Sprintf("%x", hash.Sum(nil))

		hashes = append(hashes, sha1Hash)
	}

	return hashes
}
