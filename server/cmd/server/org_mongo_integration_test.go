//go:build integration
// +build integration

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
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
		_ = db.Collection("org_domains").Drop(context.Background())
		_ = db.Collection("org_configs").Drop(context.Background())
		_ = db.Collection("org_active_config").Drop(context.Background())
		_ = db.Collection("org_submission_links").Drop(context.Background())
		_ = db.Collection("org_dpps").Drop(context.Background())
	})
	server := &Server{store: NewMongoStore(db)}
	server.ensureOrgStores()
	if _, ok := server.orgDomains.(*org.MongoDomainStore); !ok {
		t.Fatalf("domains = %T, want *org.MongoDomainStore", server.orgDomains)
	}
	if _, ok := server.orgConfigs.(*org.MongoConfigStore); !ok {
		t.Fatalf("configs = %T, want *org.MongoConfigStore", server.orgConfigs)
	}
	if _, ok := server.orgLinks.(*org.MongoLinkStore); !ok {
		t.Fatalf("links = %T, want *org.MongoLinkStore", server.orgLinks)
	}
	if _, ok := server.orgDPPs.(*org.MongoDPPStore); !ok {
		t.Fatalf("passports = %T, want *org.MongoDPPStore", server.orgDPPs)
	}
	_, err = server.orgConfigs.ActiveVersion(t.Context(), "feragame")
	if !errors.Is(err, org.ErrNotFound) {
		t.Fatalf("expected no Active Config from ensureOrgStores, got %v", err)
	}
}
