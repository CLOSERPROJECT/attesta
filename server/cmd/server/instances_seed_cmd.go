package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func runSeedInstancesCommand(ctx context.Context, args []string) error {
	_ = args
	return withSeedSession(ctx, nil, workflowConfigDir(), openMongoTaxonomyStore, nil, func(sess seedSession) error {
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
	})
}
