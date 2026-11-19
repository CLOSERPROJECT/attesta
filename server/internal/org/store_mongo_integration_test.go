//go:build integration
// +build integration

package org

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDomainAndConfigStoreRoundTrip(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	configs := NewMongoConfigStore(db)

	org := "mongo-roundtrip-" + t.Name()
	domain := domainFromKinds(map[string]EntityMap{
		"lot": {
			"NIR-1": {"client": "NO", "a.b": 42},
		},
		"kind.x": {"id.1": {"k.v": true}},
	})
	domain.Submissions = []Submission{{
		JobID: "intake", TaskID: "register", JobInstanceID: "NIR-1",
		UserID: "ada", Facts: map[string]any{"f.g": "h"},
		IDsTouched: []string{"NIR-1"},
	}}
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "NIR-1", Label: "L"}}
	if err := domains.Put(ctx, org, domain); err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["NIR-1"]["client"] != "NO" {
		t.Fatalf("load = %#v", got.Kind("lot"))
	}
	switch got.Kind("lot")["NIR-1"]["a.b"].(type) {
	case int, int32, int64, float64:
	default:
		t.Fatalf("dotted fact = %#v", got.Kind("lot")["NIR-1"]["a.b"])
	}
	if got.Kind("kind.x")["id.1"]["k.v"] != true {
		t.Fatalf("extra entity = %#v", got.Entities)
	}
	if len(got.Submissions) != 1 || got.Submissions[0].Facts["f.g"] != "h" {
		t.Fatalf("submissions = %#v", got.Submissions)
	}

	cfg := OrgConfig{
		ID:      "cfg-1",
		Org:     org,
		Version: "v1",
		Roles:   []string{"administration"},
		Jobs:    newOrderedObject(),
	}
	cfg.Jobs.Set("intake", map[string]any{"repeatable": true})
	if err := configs.Put(ctx, org, "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, org, "v1"); err != nil {
		t.Fatal(err)
	}
	loaded, err := configs.Get(ctx, org, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "v1" || loaded.Org != org {
		t.Fatalf("loaded = %#v", loaded)
	}
	intakeVal, ok := loaded.Jobs.Get("intake")
	if !ok || len(loaded.Jobs.Keys) != 1 || loaded.Jobs.Keys[0] != "intake" {
		t.Fatalf("jobs = %#v", loaded.Jobs)
	}
	if _, typed := intakeVal.(Job); !typed {
		t.Fatalf("Get should normalize Job, got %T", intakeVal)
	}
	active, err := configs.ActiveVersion(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if active != "v1" {
		t.Fatalf("active = %q", active)
	}
	if err := configs.SetActiveVersion(ctx, org, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("set active missing: %v", err)
	}
	if _, err := domains.Get(ctx, "no-such-org"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing domain: %v", err)
	}
}

func TestMongoStoresSurviveReconnect(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	org := "mongo-survive-" + t.Name()

	domains := NewMongoDomainStore(db)
	configs := NewMongoConfigStore(db)
	if err := domains.Put(ctx, org, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Org = org
	if err := configs.Put(ctx, org, cfg.Version, cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, org, cfg.Version); err != nil {
		t.Fatal(err)
	}

	// New store handles on the same DB simulate process restart with shared cluster.
	domains2 := NewMongoDomainStore(db)
	configs2 := NewMongoConfigStore(db)
	got, err := domains2.Get(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["L1"]["client"] != "NO" {
		t.Fatalf("domain after reconnect = %#v", got)
	}
	active, err := configs2.ActiveVersion(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if active != cfg.Version {
		t.Fatalf("active = %q", active)
	}
	loaded, err := configs2.Get(ctx, org, cfg.Version)
	if err != nil {
		t.Fatal(err)
	}
	intakeVal, ok := loaded.Jobs.Get("intake")
	if !ok {
		t.Fatalf("config jobs = %#v", loaded.Jobs.Keys)
	}
	if _, typed := intakeVal.(Job); !typed {
		t.Fatalf("Get should normalize Job, got %T", intakeVal)
	}
	_, task, err := lookupTask(loaded, "intake", "register")
	if err != nil {
		t.Fatalf("lookupTask after Mongo Get: %v", err)
	}
	if task == nil {
		t.Fatal("register task nil")
	}
}

func TestMongoSeedDemoWipe(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	configs := NewMongoConfigStore(db)

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(ctx, cfg.Org, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
		t.Fatal(err)
	}
	seeded, err := SeedDemo(ctx, domains, configs, SeedOptions{WipeDomain: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(ctx, seeded.Org)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Kind("truck")[SeedTruckID]; !ok {
		t.Fatalf("demo truck missing after wipe+seed: %#v", got.Kind("truck"))
	}
	if got.Kind("lot")[SeedLoadCOID] == nil || got.Kind("lot")[SeedLoadDAID] == nil {
		t.Fatalf("demo loads missing after wipe+seed: %#v", got.Kind("lot"))
	}
	if len(got.Submissions) < 8 {
		t.Fatalf("demo submissions = %#v", got.Submissions)
	}
	active, err := configs.ActiveVersion(ctx, seeded.Org)
	if err != nil {
		t.Fatal(err)
	}
	if active != DefaultFixtureVersion {
		t.Fatalf("active = %q", active)
	}
}

func appendSubmission(id string) func(Domain) (Domain, error) {
	return func(d Domain) (Domain, error) {
		d.Submissions = append(d.Submissions, Submission{JobID: "j", TaskID: "t", JobInstanceID: id})
		return d, nil
	}
}

func TestMongoDomainUpdateConcurrentAppendsAllLand(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	org := "mongo-update-concurrent"

	// Fewer writers than maxDomainUpdateAttempts so a fair retry loop cannot exhaust.
	const n = maxDomainUpdateAttempts
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := domains.Update(ctx, org, appendSubmission(string(rune('a'+i)))); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	got, err := domains.Get(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Submissions) != n {
		t.Fatalf("submissions = %d, want %d", len(got.Submissions), n)
	}
}

func TestMongoDomainUpdateLegacyDocWithoutRevision(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	org := "mongo-update-legacy"

	_, err := db.Collection(collectionOrgDomains).InsertOne(ctx, bson.M{
		"_id":          org,
		"entities":     bson.M{"lot": bson.M{"L1": bson.M{"client": "NO"}}},
		"submissions":  bson.A{},
		"jobInstances": bson.A{},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if _, err := domains.Update(ctx, org, appendSubmission(id)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := domains.Get(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Submissions) != 2 || got.Kind("lot")["L1"]["client"] != "NO" {
		t.Fatalf("domain = %#v", got)
	}
	var raw struct {
		Revision int64 `bson:"revision"`
	}
	if err := db.Collection(collectionOrgDomains).FindOne(ctx, bson.M{"_id": org}).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if raw.Revision != 2 {
		t.Fatalf("revision = %d, want 2", raw.Revision)
	}
}

func TestMongoDomainPutBumpsRevisionSoStaleUpdateRetries(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	org := "mongo-update-stale"

	if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err := domains.Update(ctx, org, func(d Domain) (Domain, error) {
		calls++
		if calls == 1 {
			// Lands between this Update's read and its conditional write.
			if err := domains.Put(ctx, org, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
				t.Fatal(err)
			}
		}
		return appendSubmission("a")(d)
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("fn calls = %d, want 2 (stale attempt retried)", calls)
	}
	got, err := domains.Get(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["L1"]["client"] != "NO" || len(got.Submissions) != 1 {
		t.Fatalf("retry must build on the Put: %#v", got)
	}
}

func TestMongoDomainUpdateConflictWhenRetriesExhausted(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	org := "mongo-update-exhausted"

	if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	_, err := domains.Update(ctx, org, func(d Domain) (Domain, error) {
		if err := domains.Put(ctx, org, EmptyDomain()); err != nil {
			t.Fatal(err)
		}
		return d, nil
	})
	if !errors.Is(err, ErrDomainConflict) {
		t.Fatalf("err = %v, want ErrDomainConflict", err)
	}
}

func livingTestMongoDB(t *testing.T) *mongo.Database {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("skip mongo store test: connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("skip mongo store test: ping: %v", err)
	}
	dbName := "living_store_test"
	if override := os.Getenv("MONGODB_DATABASE"); override != "" {
		dbName = override + "_living_store_test"
	}
	db := client.Database(dbName)
	t.Cleanup(func() {
		_ = db.Collection(collectionOrgDomains).Drop(context.Background())
		_ = db.Collection(collectionOrgConfigs).Drop(context.Background())
		_ = db.Collection(collectionOrgActiveConfig).Drop(context.Background())
	})
	return db
}
