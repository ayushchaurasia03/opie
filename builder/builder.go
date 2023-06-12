package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "os"
)

// Set Variables
var path *string
func init() {
    path = flag.String("path", "/Users/greghacke/Documents", "full path")
}

func main() {
    // Capture and string-ify path
    flag.Parse()
    basePath := *path

    err := readDirectory(basePath)
    if err != nil {
        fmt.Printf("Error reading directory: %s\n", err)
        os.Exit(1)
    }
}

func readDirectory(path string) error {
    files, err := ioutil.ReadDir(path)
    if err != nil {
        fmt.Printf("Error reading directory: %s\n", err)
        return nil
    }

    for _, file := range files {
        filePath := path + "/" + file.Name()

        if file.IsDir() {
            if err := readDirectory(filePath); err != nil {
                fmt.Printf("Error reading directory %s: %s\n", filePath, err)
            }
        } else {
            fmt.Println(filePath)
        }
    }

    return nil
}

