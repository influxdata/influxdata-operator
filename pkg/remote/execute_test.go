package remote

import (
	"fmt"
	"path"
	"testing"
)

func TestFileSpecParsing(t *testing.T) {
	result, err := extractFileSpec("default/influxdb-0:/var/lib/influxdb/backups/20180105111111")
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v\n", result)
}

func TestFileSpecPrefixCleaning(t *testing.T) {
	result, _ := extractFileSpec("default/influxdb-0:/var/lib/influxdb/backups/20180105111111")
	prefix := path.Clean(result.File)
	fmt.Println(prefix)

	// remove extraneous path shortcuts - these could occur if a path contained extra "../"
	// and attempted to navigate beyond "/" in a remote filesystem
	//prefix = stripPathShortcuts(prefix)

	fmt.Println(prefix)
}
