package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func runSeedCategoriesCommand(ctx context.Context, args []string) error {
	return seedCategoriesWithStoreOpener(ctx, args, openMongoTaxonomyStore)
}

func seedCategoriesWithStoreOpener(ctx context.Context, args []string, open func(context.Context) (Store, func(), error)) error {
	if open == nil {
		return fmt.Errorf("taxonomy seed: store opener is required")
	}
	store, cleanup, err := open(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	seedPath := taxonomySeedPathFromArgs(args)
	if err := seedTaxonomyFromFile(ctx, store, seedPath); err != nil {
		return err
	}
	log.Printf("seeded taxonomy from %s", seedPath)
	return nil
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

func taxonomySeedPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--file" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if strings.HasPrefix(arg, "--file=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--file="))
		}
	}
	if path := strings.TrimSpace(os.Getenv("CATEGORIES_SEED")); path != "" {
		return path
	}
	defaultConfigPath := envOr("WORKFLOW_CONFIG", "config/workflow.yaml")
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(defaultConfigPath)
	}
	return filepath.Join(configDir, "categories.yaml")
}
