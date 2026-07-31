package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runSeedCatalogStreamsCommand(ctx context.Context, args []string) error {
	_ = args
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return seedCatalogStreamsWithStoreOpener(ctx, nil, configDir, openMongoTaxonomyStore)
}

func seedCatalogStreamsWithStoreOpener(
	ctx context.Context,
	rng catalogStreamRNG,
	configDir string,
	open func(context.Context) (Store, func(), error),
) error {
	if open == nil {
		return fmt.Errorf("catalog stream seed: store opener is required")
	}
	store, cleanup, err := open(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	server := &Server{store: store, configDir: configDir}
	res, err := seedCatalogStreams(ctx, server, rng)
	if err != nil {
		return err
	}
	log.Printf("seeded %d catalog streams across %d taxonomy leaves; next: task seed:instances", res.Streams, res.Leaves)
	return nil
}
