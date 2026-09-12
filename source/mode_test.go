package main

import "testing"

func TestNoArgsUsesSafeProductLaunch(t *testing.T) {
	if got := resolveMode(nil); got != "--launch" {
		t.Fatalf("resolveMode(nil)=%q; want --launch", got)
	}
}

func TestWatcherRequiresExplicitRun(t *testing.T) {
	if got := resolveMode([]string{"--run"}); got != "--run" {
		t.Fatalf("explicit --run resolved to %q", got)
	}
	for _, args := range [][]string{{"foo"}, {"--report", "x.txt"}, {""}} {
		if got := resolveMode(args); got == "--run" {
			t.Fatalf("args=%q unexpectedly start resident watcher", args)
		}
	}
}

func TestKnownModes(t *testing.T) {
	for _, want := range []string{"--launch", "--self-test", "--idle-test", "--accept-install", "--install", "--uninstall", "--stop", "--status"} {
		if got := resolveMode([]string{want}); got != want {
			t.Fatalf("resolveMode(%q)=%q", want, got)
		}
	}
}
