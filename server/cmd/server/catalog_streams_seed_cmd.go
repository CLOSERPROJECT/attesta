package main

import (
	"context"
	"log"
	"math/rand"
	"time"
)

func runSeedCatalogStreamsCommand(ctx context.Context, args []string) error {
	_ = args
	return seedCatalogStreamsWithStoreOpener(ctx, nil, workflowConfigDir(), openMongoTaxonomyStore)
}

func seedCatalogStreamsWithStoreOpener(
	ctx context.Context,
	rng catalogStreamRNG,
	configDir string,
	open func(context.Context) (Store, func(), error),
) error {
	sess, err := openSeedSession(ctx, nil, configDir, open, nil)
	if err != nil {
		return err
	}
	if sess.Cleanup != nil {
		defer sess.Cleanup()
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	res, err := seedCatalogStreams(ctx, sess.Store, sess.ConfigDir, time.Now().UTC(), rng)
	if err != nil {
		return err
	}
	log.Printf("seeded %d catalog streams across %d taxonomy leaves; next: task seed:instances", res.Streams, res.Leaves)
	return nil
}
