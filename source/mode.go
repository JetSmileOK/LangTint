package main

import "strings"

func resolveMode(args []string) string {
	// Product invariant: the public EXE never self-installs. Setup is the only
	// installation entry point. A no-argument launch is safe and only starts
	// the already-installed background app (or explains which Setup to use).
	if len(args) == 0 {
		return "--launch"
	}
	for _, a := range args {
		switch strings.ToLower(a) {
		case "--launch", "--run", "--self-test", "--idle-test", "--accept-install", "--install", "--uninstall", "--stop", "--status":
			return strings.ToLower(a)
		}
	}
	return "--invalid"
}
