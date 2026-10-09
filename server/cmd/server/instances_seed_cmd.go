package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runSeedInstancesCommand(ctx context.Context, args []string) error {
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return seedInstancesWithStoreOpener(ctx, args, openMongoTaxonomyStore, configDir)
}

func seedInstancesWithStoreOpener(ctx context.Context, _ []string, open func(context.Context) (Store, func(), error), configDir string) error {
	if open == nil {
		return fmt.Errorf("instance seed: store opener is required")
	}
	store, cleanup, err := open(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	server := &Server{store: store, configDir: configDir, now: func() time.Time { return time.Now().UTC() }}
	catalog, err := server.workflowCatalog()
	if err != nil {
		return fmt.Errorf("instance seed: catalog: %w", err)
	}
	now := server.now()
	result, err := seedDemoInstances(ctx, store, catalog, now)
	if err != nil {
		return fmt.Errorf("instance seed: %w", err)
	}
	log.Printf("seeded %d instances across %d streams (%d left empty for empty-state UI)", result.Instances, result.SeededStreams, result.EmptyStreams)
	return nil
}
