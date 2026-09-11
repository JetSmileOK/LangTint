package main

import "testing"

func TestNoArgsNeverStartsWatcher(t *testing.T) {
	if got := resolveMode(nil); got != "--interactive" {
		t.Fatalf("resolveMode(nil)=%q; want --interactive", got)
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
	for _, want := range []string{"--self-test", "--idle-test", "--accept-install", "--install", "--uninstall", "--status"} {
		if got := resolveMode([]string{want}); got != want {
			t.Fatalf("resolveMode(%q)=%q", want, got)
		}
	}
}
