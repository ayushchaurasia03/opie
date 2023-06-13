package main

import (
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
	hashes := getHashes(paths)
	fmt.Println(paths)
	fmt.Println(hashes)
}

func getPathsBetween(file, root string) []string {
	var paths []string

	for {
		file = filepath.Dir(file)
		if file == root {
			break
		}
		paths = append(paths, file)
	}

	return paths
}

func getHashes(paths []string) []string {
	var hashes []string

	for _, path := range paths {

		hashes = append(hashes, path)
	}

	return hashes
}
