// Copyright 2023, RSKGroup. All rights reserved.
// Use of this source code is governed by the GNU/GPLv2 license,
// which can be found in the LICENSE file.

// Package getConfig implements a means of reading and leveraging a json file
// for configuration settings. The json file is named config.json and is
// expected to be in the same directory as the executable.
//
// The json file is expected to have the following structure:
//	{
//		"DbType": "mongodb",
//		"Host": "localhost",
//		"Port": "27017",
//		"DbUser": "user",
//		"DbPwd": "password",
//		"DbName": "database",
//		"FileColl": "files",
//		"TreeColl": "tree",
//		"NoExif": [
//			".txt",
//			".pdf"
//		],
//		"Watcher": [
//			"/home/user/Pictures",
//			"/home/user/Documents"
//		]
//	}

package getConfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

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

func getConfig() {
	// START Read JSON Config
	jsonData, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Parse the JSON data
	var varConfig Configuration
	err = json.Unmarshal(jsonData, &varConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	// END Read JSON Config

	// fmt.Println("DbType: ", varConfig.DbType)
	// fmt.Println("Host: ", varConfig.Host)
	// fmt.Println("Port: ", varConfig.Port)
	// fmt.Println("DbUser: ", varConfig.DbUser)
	// fmt.Println("DbPwd: ", varConfig.DbPwd)
	// fmt.Println("DbName: ", varConfig.DbName)
	// fmt.Println("FileColl: ", varConfig.FileColl)
	// fmt.Println("TreeColl: ", varConfig.TreeColl)
	// fmt.Println("NoExif: ", varConfig.NoExif)
	// fmt.Println("Watcher: ", varConfig.Watcher)
}
