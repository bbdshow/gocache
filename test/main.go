package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {

	fmt.Println(filenameExists("/Users"))
	filename := "/Users/hzq/workspace/go/src/gocache/test1/"
	if filenameExists(filename) {
		if err := os.Remove(filename); err != nil {
			fmt.Println(err)
		}
	} else {
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0666); err != nil {
			fmt.Println(err)
		}
	}
	// fmt.Println(os.Remove("/Users/hzq/workspace/go/src/gocache/test1/"))
}

func filenameExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
