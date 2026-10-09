package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// seedSession is the shared runtime handle for demo-seed CLIs: Mongo store,
// optional Appwrite identity, workflow config dir, and an explicit categories
// path from --file / CATEGORIES_SEED (empty means unset).
type seedSession struct {
	Store          Store
	Identity       IdentityStore
	Cleanup        func()
	ConfigDir      string
	CategoriesPath string
}

func workflowConfigDir() string {
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return configDir
}

// categoriesSeedPathFromArgs returns an explicit taxonomy path from --file or
// CATEGORIES_SEED. Empty means the caller (or Seed) should default to
// ConfigDir/categories.yaml.
func categoriesSeedPathFromArgs(args []string) string {
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

// openSeedSession opens the Mongo store (required) and optional identity, and
// resolves ConfigDir / CategoriesPath. When openIdentity is nil, Identity is
// left unset. On identity open failure, the store cleanup runs before return.
func openSeedSession(
	ctx context.Context,
	args []string,
	configDir string,
	openStore func(context.Context) (Store, func(), error),
	openIdentity func() (IdentityStore, error),
) (seedSession, error) {
	if openStore == nil {
		return seedSession{}, fmt.Errorf("seed: store opener is required")
	}
	if strings.TrimSpace(configDir) == "" {
		configDir = workflowConfigDir()
	}
	store, cleanup, err := openStore(ctx)
	if err != nil {
		return seedSession{}, err
	}
	sess := seedSession{
		Store:          store,
		Cleanup:        cleanup,
		ConfigDir:      configDir,
		CategoriesPath: categoriesSeedPathFromArgs(args),
	}
	if openIdentity == nil {
		return sess, nil
	}
	identity, err := openIdentity()
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return seedSession{}, err
	}
	sess.Identity = identity
	return sess, nil
}

func openMongoTaxonomyStore(ctx context.Context) (Store, func(), error) {
	mongoURI := envOr("MONGODB_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}
	dbName := envOr("MONGODB_DATABASE", "closer_demo")
	store := NewMongoStore(client.Database(dbName))
	cleanup := func() { _ = client.Disconnect(ctx) }
	return store, cleanup, nil
}
