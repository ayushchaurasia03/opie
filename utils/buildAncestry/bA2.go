package main

import (
	"fmt"
	"path/filepath"
)

func main() {
    file := "/Users/greghacke/Documents/Files/weights.xls"
    root := "/"
    paths := getPathsBetween(file, root)
    fmt.Println(paths)
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

