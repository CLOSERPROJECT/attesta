package main

import (
	"context"
	"log"
	"math/rand"
	"time"
)

func runSeedCatalogStreamsCommand(ctx context.Context, args []string) error {
	_ = args
	return withSeedSession(ctx, nil, workflowConfigDir(), openMongoTaxonomyStore, nil, func(sess seedSession) error {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		res, err := seedCatalogStreams(ctx, sess.Store, sess.ConfigDir, time.Now().UTC(), rng)
		if err != nil {
			return err
		}
		log.Printf("seeded %d catalog streams across %d taxonomy leaves; next: task seed:instances", res.Streams, res.Leaves)
		return nil
	})
}
