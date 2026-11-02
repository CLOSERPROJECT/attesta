package living

import (
	"context"
	"errors"
	"testing"
)

func TestEncodeDecodeMongoKey(t *testing.T) {
	if got := encodeMongoKey("1.1"); got != "1_1" {
		t.Fatalf("encode = %q", got)
	}
	if got := decodeMongoKey("1_1"); got != "1.1" {
		t.Fatalf("decode = %q", got)
	}
	facts := encodeStringKeyedMap(map[string]any{"a.b": 1, "plain": "x"})
	if _, ok := facts["a_b"]; !ok {
		t.Fatalf("encoded facts = %#v", facts)
	}
	decoded := decodeStringKeyedMap(facts)
	if decoded["a.b"] != 1 || decoded["plain"] != "x" {
		t.Fatalf("decoded = %#v", decoded)
	}
}

func TestMongoCodecAndValidationWithoutDB(t *testing.T) {
	domains := NewMongoDomainStore(nil)
	if domains == nil {
		t.Fatal("NewMongoDomainStore")
	}
	if _, err := domains.Get(context.Background(), "  "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty org get = %v", err)
	}
	if err := domains.Put(context.Background(), "", EmptyDomain()); err == nil {
		t.Fatal("empty org put")
	}

	configs := NewMongoConfigStore(nil)
	if configs == nil {
		t.Fatal("NewMongoConfigStore")
	}
	if _, err := configs.Get(context.Background(), "", "v1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty config get = %v", err)
	}
	if _, err := configs.Get(context.Background(), "org", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty version get = %v", err)
	}
	if err := configs.Put(context.Background(), "org", "", OrgConfig{}); err == nil {
		t.Fatal("empty version put")
	}
	if err := configs.Put(context.Background(), "", "v1", OrgConfig{}); err == nil {
		t.Fatal("empty org put")
	}
	if _, err := configs.ActiveVersion(context.Background(), " "); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty active = %v", err)
	}
	if err := configs.SetActiveVersion(context.Background(), "org", ""); err == nil {
		t.Fatal("empty set active")
	}
	if err := configs.SetActiveVersion(context.Background(), "", "v1"); err == nil {
		t.Fatal("empty org set active")
	}

	domain := Domain{
		Load: EntityMap{"a.b": {"k.v": 1}},
		Extra: map[string]*EntityMap{
			"kind.x": {"id.1": {"f.g": true}},
			"nil":    nil,
		},
		Submissions: []Submission{{
			JobID: "intake", TaskID: "register", JobInstanceID: "a.b",
			UserID: "ada", Facts: map[string]any{"p.q": "r"},
			IDsTouched: []string{"a.b"},
		}},
		JobInstances: []JobInstance{{JobID: "intake", ID: "a.b", Label: "L"}},
	}
	doc := domainToMongoDoc("org", domain)
	if doc.Org != "org" || doc.Load["a_b"]["k_v"] != 1 {
		t.Fatalf("encode load = %#v", doc.Load)
	}
	if doc.Extra["kind_x"]["id_1"]["f_g"] != true {
		t.Fatalf("encode extra = %#v", doc.Extra)
	}
	if _, ok := doc.Extra["nil"]; !ok {
		t.Fatal("nil extra kind missing")
	}
	if len(doc.Submissions) != 1 || doc.Submissions[0].Facts["p_q"] != "r" {
		t.Fatalf("encode submissions = %#v", doc.Submissions)
	}
	got := domainFromMongoDoc(doc)
	if got.Load["a.b"]["k.v"] != 1 {
		t.Fatalf("decode load = %#v", got.Load)
	}
	if got.Extra == nil || got.Extra["kind.x"] == nil || (*got.Extra["kind.x"])["id.1"]["f.g"] != true {
		t.Fatalf("decode extra = %#v", got.Extra)
	}
	if len(got.Submissions) != 1 || got.Submissions[0].Facts["p.q"] != "r" {
		t.Fatalf("decode submissions = %#v", got.Submissions)
	}

	emptyDoc := domainToMongoDoc("org", Domain{})
	empty := domainFromMongoDoc(emptyDoc)
	if empty.Submissions == nil || empty.JobInstances == nil {
		t.Fatalf("empty decode nils: %#v", empty)
	}
	if encodeEntityMap(nil) == nil || decodeEntityMap(nil) == nil {
		t.Fatal("nil entity maps")
	}
	if encodeSubmissions(nil) == nil || decodeSubmissions(nil) == nil {
		t.Fatal("nil submissions")
	}
	if encodeStringKeyedMap(nil) == nil || decodeStringKeyedMap(nil) == nil {
		t.Fatal("nil string-keyed maps")
	}
}
