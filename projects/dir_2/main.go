package main

import (
	"fmt"
	"os"
)

func main() {
	sourcePath := "../../../System Design Books/Systems Programming Golang.pdf"
	symlinkPath := "../dir_2/Shorcut_to_Systems_Programming"

	err := os.Symlink(sourcePath, symlinkPath)
	if err != nil {
		fmt.Printf("Error in creating symlink: %v \n", err)
		return
	}
	fmt.Printf("Symlink created: %s -> %s \n", symlinkPath, sourcePath)
}