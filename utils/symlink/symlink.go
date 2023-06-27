package main

import (
        "flag"
	"fmt"
	"os"
)

var path *string

func init() {
	path = flag.String("path", "/home/delimp/Downloads/OPIe", "full path")
}

func main() {
        flag.Parse()
        filePath := *path

	// Get file information using Lstat
	info, err := os.Lstat(filePath)
	if err != nil {
		fmt.Println("Failed to get file information: ", err)
		return
	}

	// Check if the file is a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		fmt.Printf("%s is a symbolic link\n", filePath)

		// Get the destination path of the symlink
		destPath, err := os.Readlink(filePath)
		if err != nil {
			fmt.Println("Failed to get symlink destination: ", err)
			return
		}
		fmt.Printf("Symlink destination: %s\n", destPath)
	} else {
		fmt.Printf("%s is not a symbolic link\n", filePath)
	}
}

