package main

import (
	"context"
	"log"
	"os"
)

func runBootstrapAppwriteCommand(ctx context.Context, _ []string) error {
	deps := bootstrapAppwriteDepsFromEnv()
	result, err := BootstrapAppwriteProject(ctx, deps)
	if err != nil {
		return err
	}
	// Make credentials visible to this process and any later same-shell steps.
	_ = os.Setenv("APPWRITE_PROJECT_ID", result.ProjectID)
	_ = os.Setenv("APPWRITE_API_KEY", result.APIKey)
	switch {
	case result.Skipped:
		log.Printf("bootstrap-appwrite: existing API key works; wrote %s", deps.EnvFilePath)
	case result.Created:
		log.Printf("bootstrap-appwrite: provisioned project %q; wrote %s", result.ProjectID, deps.EnvFilePath)
	default:
		log.Printf("bootstrap-appwrite: wrote %s", deps.EnvFilePath)
	}
	return nil
}
