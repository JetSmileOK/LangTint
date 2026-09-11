package main

import "strings"

func resolveMode(args []string) string {
	// Safety invariant: no arguments means interactive acceptance/install,
	// never a resident watcher from the source/download directory.
	if len(args) == 0 {
		return "--interactive"
	}
	for _, a := range args {
		switch strings.ToLower(a) {
		case "--run", "--self-test", "--idle-test", "--accept-install", "--install", "--uninstall", "--status":
			return strings.ToLower(a)
		}
	}
	return "--invalid"
}
