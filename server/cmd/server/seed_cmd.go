package main

import (
	"context"
)

func runSeedCommand(ctx context.Context, args []string) error {
	return runSeedSession(ctx, args, seedReplaceFromArgs(args))
}

func seedReplaceFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--replace" {
			return true
		}
	}
	return false
}
