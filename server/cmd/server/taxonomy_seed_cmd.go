package main

import (
	"context"
	"log"
	"path/filepath"
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
	seedPath := sess.CategoriesPath
	if seedPath == "" {
		seedPath = filepath.Join(sess.ConfigDir, "categories.yaml")
	}
	if err := seedTaxonomyFromFile(ctx, sess.Store, seedPath); err != nil {
		return err
	}
	log.Printf("seeded taxonomy from %s", seedPath)
	return nil
}

// taxonomySeedPathFromArgs resolves the taxonomy CLI seed file: --file /
// CATEGORIES_SEED, else ConfigDir/categories.yaml.
func taxonomySeedPathFromArgs(args []string) string {
	if path := categoriesSeedPathFromArgs(args); path != "" {
		return path
	}
	return filepath.Join(workflowConfigDir(), "categories.yaml")
}
