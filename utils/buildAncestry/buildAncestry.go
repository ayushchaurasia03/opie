package buildAncestry

import (
	"flag"
)

var file *string
var root *string

func init() {
	file = flag.String("file", "/pat/to/your/file/name.txt", "full file path")
	root := flag.String("root", "/pat/to/your/file/", "root path")
}
