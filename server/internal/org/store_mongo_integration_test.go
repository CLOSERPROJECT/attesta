//go:build integration
// +build integration

package org

import (
	"context"
	"errors"
	"os"
	"slices"
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
		EntitiesTouched: []EntityTouch{{Op: "mint", Entity: "lot", ID: "NIR-1"}},
	}}
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "NIR-1", Label: "L"}}
	if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, domain); err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(ctx, Scope{Org: org, Stream: "main"})
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
	if err := configs.Put(ctx, Scope{Org: org, Stream: "main"}, "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, Scope{Org: org, Stream: "main"}, "v1"); err != nil {
		t.Fatal(err)
	}
	loaded, err := configs.Get(ctx, Scope{Org: org, Stream: "main"}, "v1")
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
	active, err := configs.ActiveVersion(ctx, Scope{Org: org, Stream: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if active != "v1" {
		t.Fatalf("active = %q", active)
	}
	if err := configs.SetActiveVersion(ctx, Scope{Org: org, Stream: "main"}, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("set active missing: %v", err)
	}
	if _, err := domains.Get(ctx, Scope{Org: "no-such-org", Stream: "main"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing domain: %v", err)
	}
}

func TestMongoStoresSurviveReconnect(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	org := "mongo-survive-" + t.Name()

	domains := NewMongoDomainStore(db)
	configs := NewMongoConfigStore(db)
	if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Org = org
	if err := configs.Put(ctx, Scope{Org: org, Stream: "main"}, cfg.Version, cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, Scope{Org: org, Stream: "main"}, cfg.Version); err != nil {
		t.Fatal(err)
	}

	// New store handles on the same DB simulate process restart with shared cluster.
	domains2 := NewMongoDomainStore(db)
	configs2 := NewMongoConfigStore(db)
	got, err := domains2.Get(ctx, Scope{Org: org, Stream: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["L1"]["client"] != "NO" {
		t.Fatalf("domain after reconnect = %#v", got)
	}
	active, err := configs2.ActiveVersion(ctx, Scope{Org: org, Stream: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if active != cfg.Version {
		t.Fatalf("active = %q", active)
	}
	loaded, err := configs2.Get(ctx, Scope{Org: org, Stream: "main"}, cfg.Version)
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
	if err := domains.Put(ctx, Scope{Org: cfg.Org, Stream: cfg.Stream.Key}, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
		t.Fatal(err)
	}
	seeded, err := SeedDemo(ctx, domains, configs, SeedOptions{WipeDomain: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(ctx, Scope{Org: seeded.Org, Stream: seeded.Stream.Key})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Kind("truck")[SeedTruckID]; !ok {
		t.Fatalf("demo truck A missing after wipe+seed: %#v", got.Kind("truck"))
	}
	if _, ok := got.Kind("truck")[SeedTruckBID]; !ok {
		t.Fatalf("demo truck B missing after wipe+seed: %#v", got.Kind("truck"))
	}
	if got.Kind("lot")[SeedLoadCOID] == nil || got.Kind("lot")[SeedLoadDAID] == nil || got.Kind("lot")[SeedTruckBLotID] == nil {
		t.Fatalf("demo loads missing after wipe+seed: %#v", got.Kind("lot"))
	}
	if len(got.Submissions) < 8 {
		t.Fatalf("demo submissions = %#v", got.Submissions)
	}
	active, err := configs.ActiveVersion(ctx, Scope{Org: seeded.Org, Stream: seeded.Stream.Key})
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
			if _, err := domains.Update(ctx, Scope{Org: org, Stream: "main"}, appendSubmission(string(rune('a'+i)))); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	got, err := domains.Get(ctx, Scope{Org: org, Stream: "main"})
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
	scope := Scope{Org: "mongo-update-legacy", Stream: "main"}
	id := scopeKey(scope)

	_, err := db.Collection(collectionOrgDomains).InsertOne(ctx, bson.M{
		"_id":          id,
		"org":          scope.Org,
		"stream":       scope.Stream,
		"entities":     bson.M{"lot": bson.M{"L1": bson.M{"client": "NO"}}},
		"submissions":  bson.A{},
		"jobInstances": bson.A{},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, sid := range []string{"a", "b"} {
		if _, err := domains.Update(ctx, scope, appendSubmission(sid)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := domains.Get(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Submissions) != 2 || got.Kind("lot")["L1"]["client"] != "NO" {
		t.Fatalf("domain = %#v", got)
	}
	var raw struct {
		Revision int64 `bson:"revision"`
	}
	if err := db.Collection(collectionOrgDomains).FindOne(ctx, bson.M{"_id": id}).Decode(&raw); err != nil {
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

	if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err := domains.Update(ctx, Scope{Org: org, Stream: "main"}, func(d Domain) (Domain, error) {
		calls++
		if calls == 1 {
			// Lands between this Update's read and its conditional write.
			if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})); err != nil {
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
	got, err := domains.Get(ctx, Scope{Org: org, Stream: "main"})
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

	if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	_, err := domains.Update(ctx, Scope{Org: org, Stream: "main"}, func(d Domain) (Domain, error) {
		if err := domains.Put(ctx, Scope{Org: org, Stream: "main"}, EmptyDomain()); err != nil {
			t.Fatal(err)
		}
		return d, nil
	})
	if !errors.Is(err, ErrDomainConflict) {
		t.Fatalf("err = %v, want ErrDomainConflict", err)
	}
}

func TestMongoStreamsOfOneOrgKeepSeparateDomainsAndActiveVersions(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	domains := NewMongoDomainStore(db)
	configs := NewMongoConfigStore(db)

	recycling, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other, err := LoadSecondStreamConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	org := "mongo-two-streams-" + t.Name()
	recycling.Org = org
	other.Org = org
	recycling.Version = "recycle-v1"
	other.Version = "other-v1"

	scopeA, err := ScopeFromConfig(recycling)
	if err != nil {
		t.Fatal(err)
	}
	scopeB, err := ScopeFromConfig(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(ctx, scopeA, recycling.Version, recycling); err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(ctx, scopeB, other.Version, other); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, scopeA, recycling.Version); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(ctx, scopeB, other.Version); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(ctx, scopeA, domainFromKinds(map[string]EntityMap{"truck": {"T-A": {"mark": "a"}}})); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(ctx, scopeB, domainFromKinds(map[string]EntityMap{"item": {"I-B": {"mark": "b"}}})); err != nil {
		t.Fatal(err)
	}

	streams, err := configs.ListStreams(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	wantStreams := []string{scopeA.Stream, scopeB.Stream}
	slices.Sort(wantStreams)
	if !slices.Equal(streams, wantStreams) {
		t.Fatalf("ListStreams = %v, want %v", streams, wantStreams)
	}
	activeA, err := configs.ActiveVersion(ctx, scopeA)
	if err != nil || activeA != recycling.Version {
		t.Fatalf("active A = %q err=%v", activeA, err)
	}
	activeB, err := configs.ActiveVersion(ctx, scopeB)
	if err != nil || activeB != other.Version {
		t.Fatalf("active B = %q err=%v", activeB, err)
	}
	gotA, err := domains.Get(ctx, scopeA)
	if err != nil || gotA.Kind("truck")["T-A"]["mark"] != "a" || len(gotA.Kind("item")) != 0 {
		t.Fatalf("domain A = %#v err=%v", gotA, err)
	}
	gotB, err := domains.Get(ctx, scopeB)
	if err != nil || gotB.Kind("item")["I-B"]["mark"] != "b" || len(gotB.Kind("truck")) != 0 {
		t.Fatalf("domain B = %#v err=%v", gotB, err)
	}
}

func TestMongoStreamsOfOneOrgKeepSeparateDPPsAndLinks(t *testing.T) {
	db := livingTestMongoDB(t)
	ctx := t.Context()
	passports := NewMongoDPPStore(db)
	links := NewMongoLinkStore(db)

	scopeA := Scope{Org: "mongo-dpp-" + t.Name(), Stream: "pv-recovery"}
	scopeB := Scope{Org: scopeA.Org, Stream: "other"}
	addr := SubmissionAddress{JobID: "intake", JobInstanceID: "T1", TaskID: "register"}
	ts := "2026-11-19T12:00:00.000000000Z"

	if err := passports.Create(ctx, DPPRecord{
		Org: scopeA.Org, Stream: scopeA.Stream, SubjectID: "L-A",
		GTIN: "09506000134352", Lot: "LOT-A", Serial: "L-A",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(ctx, DPPRecord{
		Org: scopeB.Org, Stream: scopeB.Stream, SubjectID: "L-A",
		GTIN: "09506000134352", Lot: "LOT-B", Serial: "L-A",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/b",
	}); err != nil {
		t.Fatal(err)
	}
	gotA, err := passports.Get(ctx, scopeA, "L-A")
	if err != nil || gotA.Lot != "LOT-A" || gotA.Stream != scopeA.Stream {
		t.Fatalf("passport A = %#v err=%v", gotA, err)
	}
	gotB, err := passports.Get(ctx, scopeB, "L-A")
	if err != nil || gotB.Lot != "LOT-B" || gotB.Stream != scopeB.Stream {
		t.Fatalf("passport B = %#v err=%v", gotB, err)
	}

	dup := DPPRecord{
		Org: scopeB.Org, Stream: scopeB.Stream, SubjectID: "L-dup",
		GTIN: "09506000134352", Lot: "LOT-A", Serial: "L-A",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/dup",
	}
	if err := passports.Create(ctx, dup); !errors.Is(err, ErrDPPExists) {
		t.Fatalf("cross-stream identity = %v, want ErrDPPExists", err)
	}
	byID, err := passports.GetByIdentity(ctx, "09506000134352", "LOT-A", "L-A")
	if err != nil || byID.Stream != scopeA.Stream {
		t.Fatalf("GetByIdentity = %#v err=%v", byID, err)
	}

	if err := links.Put(ctx, scopeA, addr, ts, []HttpLinkItem{{Label: "A", URL: "blob:a", Action: HttpActionLink}}); err != nil {
		t.Fatal(err)
	}
	if err := links.Put(ctx, scopeB, addr, ts, []HttpLinkItem{{Label: "B", URL: "blob:b", Action: HttpActionDownload, Filename: "b.csv"}}); err != nil {
		t.Fatal(err)
	}
	storedA, err := links.Get(ctx, scopeA, addr, ts)
	if err != nil || len(storedA) != 1 || storedA[0].URL != "blob:a" {
		t.Fatalf("links A = %#v err=%v", storedA, err)
	}
	storedB, err := links.Get(ctx, scopeB, addr, ts)
	if err != nil || len(storedB) != 1 || storedB[0].Filename != "b.csv" {
		t.Fatalf("links B = %#v err=%v", storedB, err)
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
		_ = db.Collection(collectionOrgDPPs).Drop(context.Background())
		_ = db.Collection(collectionOrgSubmissionLinks).Drop(context.Background())
	})
	return db
}
