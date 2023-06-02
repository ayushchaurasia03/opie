# OPIe
Operations Intelligence engine Utilities and Shared Packages

## utils

### utils/buildAncestry
Given a full file path to a file and the root of the monitoring path, create an array containing the full path names of parents and an array of those parent's hashes.

```
SourceFile =: /Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER/RSKGroup_Customer_Support/Apple/RetailMarketingProduction_\(RMP\)/Mini\ Migration/apple-exporters.dmg 
WatchRoot =: /Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup

"AncestryPaths":[
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER/RSKGroup_Customer_Support/Apple/RetailMarketingProduction (RMP)/Mini Migration",
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER/RSKGroup_Customer_Support/Apple/RetailMarketingProduction (RMP)",
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER/RSKGroup_Customer_Support/Apple",
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER/RSKGroup_Customer_Support",
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup/RSKGROUP_FOLDER",
	"/Users/greghacke/Library/CloudStorage/Dropbox-RSKGroup"
]

"AncestryHash": [
	"b4b29b27ed6658194f69a85d3916b0a7b084b991",
	"ccba3dc6d496b91c441e29ab6d30bd219ef0253d",
	"29302dbf47feb2e7025d39838ea136c5ce273e23",
	"ac95eb7da41f3ec1164ea76aa94fc12f7b5fdf7f",
	"cf9d23ef90016b97523fa1be25c42f286aa481a9",
	"2ab67b608e7613dba96eda7ac310108cc9e4b645"
]
```
### utils/getConfig
This utility will load the config.json into memory

### utils/getFileData
This utility is designed to get the file data from the file system using LStat and FileInfo from the default Go Packages

### utils/getFileExifData
This utility calls the OS-installed EXIFTOOL to gather additional exif data based on the file extension

### utils/flatJson
This is a fork of the https://pkg.go.dev/github.com/pushrax/flatjson package and will be modified as necessary.
#### Types
##### type Map
`type Map map[string]interface{}`
##### func Flatten
`func Flatten(val interface{}) Map`

### utils/fsnotify/fsnotify
Drawn from `github.com/fsnotify/fsnotify` to perform watcher functions. Note we thread through fsnotify to create watchers for each subfolder at initiation.

### utils/mongoWrite
Drawn from our exiting mongoDB solutions, this package in turn leverages `go.mongodb.org/mongo-driver/mongo` and `go.mongodb.org/mongo-driver/mongo/options` to properly marshal our content into a functional form and write it to MongoDB

### utils/solrWrite
Drawn from our exiting mongoDB solutions, this package in turn leverages `github.com/vanng822/go-solr` to properly marshal our content into a functional form and write it to MongoDB
