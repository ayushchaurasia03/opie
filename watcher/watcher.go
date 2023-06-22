package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var watcher *fsnotify.Watcher
var paths []string
var fileCollection *mongo.Collection

func init() {
	confPath := flag.String("conf", "conf.json", "Path to the configuration file")
	flag.Parse()

	loadConfig(*confPath)
}

type Configuration struct {
	DbType   string   `json:"DbType"`
	Host     string   `json:"Host"`
	Port     string   `json:"Port"`
	DbUser   string   `json:"DbUser"`
	DbPwd    string   `json:"DbPwd"`
	DbName   string   `json:"DbName"`
	FileColl string   `json:"FileColl"`
	TreeColl string   `json:"TreeColl"`
	NoExif   []string `json:"NoExif"`
	Watcher  []string `json:"Watcher"`
}

func loadConfig(path string) {
	// Read the configuration file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read configuration file: %v", err)
	}

	// Parse the JSON configuration
	var config Configuration
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %v", err)
	}

	// Set the watch directories
	paths = config.Watcher

	// Configure MongoDB connection
	dbType := config.DbType
	host := config.Host
	port := config.Port
	dbUser := config.DbUser
	dbPwd := config.DbPwd
	dbName := config.DbName

	// Connect to MongoDB
	err = connectToMongoDB(dbType, host, port, dbUser, dbPwd, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
}

// main
func main() {
	// create your file with desired read/write permissions
	f, err := os.OpenFile("tracelog.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// set output of logs to f
	log.SetOutput(f)

	// creates a new file watcher
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error creating watcher:", err)
	}
	defer watcher.Close()

	// watch the specified directories
	for _, path := range paths {
		err := watchDir(path)
		if err != nil {
			log.Println("ERROR", err)
		}
	}

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// Handle the events
				if event.Op&fsnotify.Create == fsnotify.Create {
					file, err := os.Stat(event.Name)
					if err != nil {
						log.Println("Error getting file info:", err)
						continue
					}
					if !file.IsDir() {
						fileInfo := struct {
							Root string `json:"Root"`
							Name string `json:"Name"`
							Date string `json:"Date"`
						}{
							Root: filepath.Dir(event.Name),
							Name: event.Name,
							Date: file.ModTime().String(),
						}
						fileInfoJSON, err := json.Marshal(fileInfo)
						if err != nil {
							log.Println("Error marshaling JSON:", err)
							continue
						}
						log.Println(string(fileInfoJSON))

						// Insert document into MongoDB
						_, err = fileCollection.InsertOne(context.TODO(), fileInfo)
						if err != nil {
							log.Println("Error inserting document into MongoDB:", err)
						}
					} else {
						dirInfo := struct {
							Root string `json:"Root"`
							Name string `json:"Name"`
							Date string `json:"Date"`
						}{
							Root: filepath.Dir(event.Name),
							Name: event.Name,
							Date: file.ModTime().String(),
						}
						dirInfoJSON, err := json.Marshal(dirInfo)
						if err != nil {
							log.Println("Error marshaling JSON:", err)
							continue
						}
						log.Println(string(dirInfoJSON))

						// Insert document into MongoDB
						_, err = fileCollection.InsertOne(context.TODO(), dirInfo)
						if err != nil {
							log.Println("Error inserting document into MongoDB:", err)
						}
					}
				}

				// ... Handle other events if needed ...
			case err := <-watcher.Errors:
				log.Println("ERROR", err)
			}
		}
	}()

	<-done
}

// watchDir gets run as a walk func, searching for directories to add watchers to
func watchDir(path string) error {
	// Add watcher for the current directory
	if err := watcher.Add(path); err != nil {
		log.Println("Error adding watcher to directory:", err)
		return err
	}

	// Continue walking only if the item is a directory
	fi, err := os.Stat(path)
	if err != nil {
		log.Println("Error getting file info:", err)
		return nil
	}
	if !fi.IsDir() {
		return nil
	}

	// Walk the nested directories and add watchers to them
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Error reading directory:", err)
		return nil
	}

	for _, file := range files {
		if file.IsDir() {
			subDirPath := filepath.Join(path, file.Name())
			if err := watchDir(subDirPath); err != nil {
				log.Println("Error adding watcher to subdirectory:", err)
			}
		}
	}

	return nil
}

func connectToMongoDB(dbType, host, port, dbUser, dbPwd, dbName string) error {
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
	fileCollection = db.Collection("watcher")

	return nil
}
