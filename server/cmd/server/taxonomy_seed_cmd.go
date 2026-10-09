package main

import (
	"context"
	"log"
)

func runSeedCategoriesCommand(ctx context.Context, args []string) error {
	return seedCategoriesWithStoreOpener(ctx, args, openMongoTaxonomyStore)
}

func seedCategoriesWithStoreOpener(ctx context.Context, args []string, open func(context.Context) (Store, func(), error)) error {
	sess, err := openSeedSession(ctx, args, "", open, nil)
	if err != nil {
		return err
	}
	if sess.Cleanup != nil {
		defer sess.Cleanup()
	}
	seedPath := resolveCategoriesSeedPath(sess.CategoriesPath, sess.ConfigDir)
	if err := seedTaxonomyFromFile(ctx, sess.Store, seedPath); err != nil {
		return err
	}
	log.Printf("seeded taxonomy from %s", seedPath)
	return nil
}
