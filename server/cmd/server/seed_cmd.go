package main

import (
	"context"
	"fmt"
	"log"
)

func runSeedCommand(ctx context.Context, args []string) error {
	return seedWithOpeners(ctx, args, workflowConfigDir(), openMongoTaxonomyStore, openAppwriteIdentity)
}

func seedWithOpeners(
	ctx context.Context,
	args []string,
	configDir string,
	openStore func(context.Context) (Store, func(), error),
	openIdentity func() (IdentityStore, error),
) error {
	if openIdentity == nil {
		return fmt.Errorf("seed: identity opener is required")
	}
	sess, err := openSeedSession(ctx, args, configDir, openStore, openIdentity)
	if err != nil {
		return err
	}
	if sess.Cleanup != nil {
		defer sess.Cleanup()
	}
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
}

func openAppwriteIdentity() (IdentityStore, error) {
	return appwriteIdentityFromEnv(appwriteIdentityFromEnvOptions{RequireAll: true})
}

func seedReplaceFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--replace" {
			return true
		}
	}
	return false
}
