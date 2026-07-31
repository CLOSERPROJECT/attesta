package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
	if len(catalog) == 0 {
		return fmt.Errorf("instance seed: workflow catalog is empty")
	}
	keys := make([]string, 0, len(catalog))
	for k := range catalog {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	now := time.Now().UTC()
	if server.now != nil {
		now = server.now()
	}
	total := 0
	empty := 0
	for i, key := range keys {
		if seedStreamLeftEmpty(i) {
			if err := store.DeleteWorkflowData(ctx, key); err != nil {
				return fmt.Errorf("seed instances %s: delete: %w", key, err)
			}
			empty++
			continue
		}
		n, err := seedWorkflowInstances(ctx, store, key, catalog[key], now)
		if err != nil {
			return err
		}
		total += n
	}
	log.Printf("seeded %d instances across %d streams (%d left empty for empty-state UI)", total, len(keys)-empty, empty)
	return nil
}
