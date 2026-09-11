//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("TaskbarLayoutTint is Windows-only. This build is for sandbox logic tests only.")
}
