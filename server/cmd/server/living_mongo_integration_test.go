//go:build integration
// +build integration

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestEnsureLivingStoresUsesMongoWhenAvailable(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("skip: mongo unavailable: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("skip: mongo ping failed: %v", err)
	}
	db := client.Database("living_wiring_test")
	t.Cleanup(func() {
		_ = db.Collection("living_domains").Drop(context.Background())
		_ = db.Collection("living_org_configs").Drop(context.Background())
		_ = db.Collection("living_active_config").Drop(context.Background())
	})
	server := &Server{store: NewMongoStore(db)}
	server.ensureLivingStores()
	if _, ok := server.livingDomains.(*living.MongoDomainStore); !ok {
		t.Fatalf("domains = %T, want *living.MongoDomainStore", server.livingDomains)
	}
	if _, ok := server.livingConfigs.(*living.MongoConfigStore); !ok {
		t.Fatalf("configs = %T, want *living.MongoConfigStore", server.livingConfigs)
	}
	_, err = server.livingConfigs.ActiveVersion(t.Context(), "feragame")
	if !errors.Is(err, living.ErrNotFound) {
		t.Fatalf("expected no Active Config from ensureLivingStores, got %v", err)
	}
}
