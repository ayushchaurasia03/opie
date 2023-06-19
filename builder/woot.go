package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	// Path to the folder
	folderPath := "/path/to/folder"

	// Get the parent directory of the folder
	parentDir := filepath.Dir(folderPath)

	fmt.Println("Parent Directory:", parentDir)
}

