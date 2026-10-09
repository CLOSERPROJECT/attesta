package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func runSeedCommand(ctx context.Context, args []string) error {
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return seedWithOpeners(ctx, args, configDir, openMongoTaxonomyStore, openAppwriteIdentity)
}

func seedWithOpeners(
	ctx context.Context,
	args []string,
	configDir string,
	openStore func(context.Context) (Store, func(), error),
	openIdentity func() (IdentityStore, error),
) error {
	if openStore == nil {
		return fmt.Errorf("seed: store opener is required")
	}
	if openIdentity == nil {
		return fmt.Errorf("seed: identity opener is required")
	}
	store, cleanup, err := openStore(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	identity, err := openIdentity()
	if err != nil {
		return err
	}
	replace := seedReplaceFromArgs(args)
	if err := Seed(ctx, SeedDeps{
		Identity:       identity,
		Store:          store,
		ConfigDir:      configDir,
		CategoriesPath: seedCategoriesPathFromArgs(args),
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
	endpoint := strings.TrimSpace(os.Getenv("APPWRITE_ENDPOINT"))
	projectID := strings.TrimSpace(os.Getenv("APPWRITE_PROJECT_ID"))
	apiKey := strings.TrimSpace(os.Getenv("APPWRITE_API_KEY"))
	if endpoint == "" || projectID == "" || apiKey == "" {
		return nil, fmt.Errorf("seed: APPWRITE_ENDPOINT, APPWRITE_PROJECT_ID, and APPWRITE_API_KEY are required")
	}
	return NewAppwriteIdentity(endpoint, projectID, apiKey, nil), nil
}

func seedReplaceFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--replace" {
			return true
		}
	}
	return false
}

// seedCategoriesPathFromArgs returns an explicit taxonomy path from --file or
// CATEGORIES_SEED. Empty means Seed should use ConfigDir/categories.yaml.
func seedCategoriesPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--file" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if strings.HasPrefix(arg, "--file=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--file="))
		}
	}
	return strings.TrimSpace(os.Getenv("CATEGORIES_SEED"))
}
