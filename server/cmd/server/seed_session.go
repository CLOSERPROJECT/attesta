package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type seedSession struct {
	Store          Store
	Identity       IdentityStore
	Cleanup        func()
	ConfigDir      string
	CategoriesPath string
}

func workflowConfigDir() string {
	return "config"
}

// Empty means Seed should default to ConfigDir/categories.yaml.
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
	return ""
}

func resolveCategoriesSeedPath(explicit, configDir string) string {
	if path := strings.TrimSpace(explicit); path != "" {
		return path
	}
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}
	return filepath.Join(dir, "categories.yaml")
}

// On identity open failure, store cleanup runs before return.
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

func withSeedSession(
	ctx context.Context,
	args []string,
	configDir string,
	openStore func(context.Context) (Store, func(), error),
	openIdentity func() (IdentityStore, error),
	fn func(seedSession) error,
) error {
	sess, err := openSeedSession(ctx, args, configDir, openStore, openIdentity)
	if err != nil {
		return err
	}
	if sess.Cleanup != nil {
		defer sess.Cleanup()
	}
	return fn(sess)
}

// creds nil → ambient identity (standalone seed applies the runtime env file);
// non-nil → prepare→seed by value (skip file apply; main is the Server restart adapter).
func runSeedSession(ctx context.Context, args []string, replace bool, creds *appwriteRuntimeCredentials) error {
	if creds == nil {
		if err := applyAppwriteRuntimeEnvFile(resolveAppwriteRuntimeEnvFile()); err != nil {
			return err
		}
	}
	return withSeedSession(ctx, args, workflowConfigDir(), openMongoTaxonomyStore, func() (IdentityStore, error) {
		opts := appwriteIdentityFromEnvOptions{RequireAll: true}
		if creds == nil {
			return appwriteIdentityFromEnv(opts)
		}
		return appwriteIdentityFromCredentials(*creds, opts)
	}, func(sess seedSession) error {
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

func openMongoStore(ctx context.Context) (Store, string, func(), error) {
	mongoURI := envOr("MONGODB_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, "", nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, "", nil, fmt.Errorf("mongo ping: %w", err)
	}
	dbName := envOr("MONGODB_DATABASE", "closer_demo")
	store := NewMongoStore(client.Database(dbName))
	cleanup := func() { _ = client.Disconnect(ctx) }
	return store, workflowConfigDir(), cleanup, nil
}

// ConfigDir is resolved separately via workflowConfigDir; openers only need Store + cleanup.
func openMongoTaxonomyStore(ctx context.Context) (Store, func(), error) {
	store, _, cleanup, err := openMongoStore(ctx)
	return store, cleanup, err
}
