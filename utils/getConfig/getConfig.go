package getConfig

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Configuration struct {
	DbType   string `json:"DbType"`
	Host     string `json:"Host"`
	Port     string `json:"Port"`
	DbUser   string `json:"DbUser"`
	DbPwd    string `json:"DbPwd"`
	DbName   string `json:"DbName"`
	FileColl string `json:"FileColl"`
	TreeColl string `json:"TreeColl"`
}

func getConfig() {
	// START Read JSON Config
	jsonData, err := ioutil.ReadFile("conf.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Parse the JSON data
	var varConf Configuration
	err = json.Unmarshal(jsonData, &varConf)
	if err != nil {
		fmt.Println(err)
		return
	}
	// END Read JSON Config
	}
