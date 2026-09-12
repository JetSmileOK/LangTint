//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("LangTint is Windows-only. This build is for sandbox logic tests only.")
}
