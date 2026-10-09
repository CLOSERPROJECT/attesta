package main

import (
	"context"
	"log"
)

func runSeedCategoriesCommand(ctx context.Context, args []string) error {
	return withSeedSession(ctx, args, "", openMongoTaxonomyStore, nil, func(sess seedSession) error {
		seedPath := resolveCategoriesSeedPath(sess.CategoriesPath, sess.ConfigDir)
		if err := seedTaxonomyFromFile(ctx, sess.Store, seedPath); err != nil {
			return err
		}
		log.Printf("seeded taxonomy from %s", seedPath)
		return nil
	})
}
