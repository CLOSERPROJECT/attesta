package main

import (
	"context"
	"log"
)

func runSeedCommand(ctx context.Context, args []string) error {
	return withSeedSession(ctx, args, workflowConfigDir(), openMongoTaxonomyStore, func() (IdentityStore, error) {
		return appwriteIdentityFromEnv(appwriteIdentityFromEnvOptions{RequireAll: true})
	}, func(sess seedSession) error {
		replace := seedReplaceFromArgs(args)
		if err := Seed(ctx, SeedDeps{
			Identity:       sess.Identity,
			Store:          sess.Store,
			ConfigDir:      sess.ConfigDir,
			CategoriesPath: sess.CategoriesPath,
			Replace:        replace,
		}); err != nil {
			return err
		}
		if replace {
			log.Printf("seed: replaced demo fixture")
		} else {
			log.Printf("seed: demo fixture ready (empty-only)")
		}
		return nil
	})
}

func seedReplaceFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--replace" {
			return true
		}
	}
	return false
}
