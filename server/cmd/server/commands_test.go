package main

import (
	"context"
	"testing"
)

func TestWithSeedRunner(t *testing.T) {
	called := false
	deps := (PrepareDeps{}).withSeedRunner(func(context.Context, appwriteRuntimeCredentials) error {
		called = true
		return nil
	})
	if deps.Seed == nil {
		t.Fatal("seed runner was not set")
	}
	if err := deps.Seed(context.Background(), appwriteRuntimeCredentials{}); err != nil {
		t.Fatalf("seed runner = %v", err)
	}
	if !called {
		t.Fatal("seed runner was not called")
	}
}

func TestRunPrepareCommandRejectsInvalidArgs(t *testing.T) {
	if err := runPrepareCommand(context.Background(), []string{"positional"}); err == nil {
		t.Fatal("positional argument must be rejected")
	}
	if err := runPrepareCommand(context.Background(), []string{"--unknown"}); err == nil {
		t.Fatal("unknown flag must be rejected")
	}
}

func TestSeedHelpFromArgs(t *testing.T) {
	if seedHelpFromArgs(nil) {
		t.Fatal("no arguments must not request help")
	}
	if seedHelpFromArgs([]string{"--replace"}) {
		t.Fatal("other flags must not request help")
	}
	if !seedHelpFromArgs([]string{"--replace", "--help"}) {
		t.Fatal("--help must request help")
	}
	if !seedHelpFromArgs([]string{"-h"}) {
		t.Fatal("-h must request help")
	}
}

func TestRunSeedCommandHelp(t *testing.T) {
	if err := runSeedCommand(context.Background(), []string{"--help"}); err != nil {
		t.Fatalf("seed help = %v", err)
	}
}
