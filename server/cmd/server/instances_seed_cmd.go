package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func runSeedInstancesCommand(ctx context.Context, args []string) error {
	return seedInstancesWithStoreOpener(ctx, args, openMongoTaxonomyStore, workflowConfigDir())
}

func seedInstancesWithStoreOpener(ctx context.Context, _ []string, open func(context.Context) (Store, func(), error), configDir string) error {
	sess, err := openSeedSession(ctx, nil, configDir, open, nil)
	if err != nil {
		return err
	}
	if sess.Cleanup != nil {
		defer sess.Cleanup()
	}
	server := &Server{store: sess.Store, configDir: sess.ConfigDir, now: func() time.Time { return time.Now().UTC() }}
	catalog, err := server.workflowCatalog()
	if err != nil {
		return fmt.Errorf("instance seed: catalog: %w", err)
	}
	now := server.now()
	result, err := seedDemoInstances(ctx, sess.Store, catalog, now)
	if err != nil {
		return fmt.Errorf("instance seed: %w", err)
	}
	log.Printf("seeded %d instances across %d streams (%d left empty for empty-state UI)", result.Instances, result.SeededStreams, result.EmptyStreams)
	return nil
}
