package living

import (
	"errors"
	"testing"
)

func TestSeedDemoSetsActiveConfigWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	seeded := Domain{
		Load: EntityMap{"L1": {"client": "NO"}},
		Submissions: []Submission{
			{JobID: "intake", TaskID: "register", JobInstanceID: "L1", UserID: "ada"},
		},
	}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	version, err := configs.ActiveVersion(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if version != DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}

	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Load) != 1 || len(got.Submissions) != 1 {
		t.Fatalf("domain wiped without WipeDomain: %#v", got)
	}
}

func TestSeedDemoWipesDomainAndSeedsFixture(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if err := domains.Put(t.Context(), "feragame", Domain{
		Load:        EntityMap{"L1": {"client": "NO"}},
		Submissions: []Submission{{JobID: "intake", TaskID: "register"}},
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: true})
	if err != nil {
		t.Fatal(err)
	}

	got, err := domains.Get(t.Context(), cfg.Org)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Load) != 0 || len(got.Submissions) != 0 || len(got.JobInstances) != 0 {
		t.Fatalf("domain not empty after wipe: %#v", got)
	}

	loader := &Loader{Domains: domains, Configs: configs}
	loaded, err := loader.Load(t.Context(), cfg.Org)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveConfigVersion != DefaultFixtureVersion {
		t.Fatalf("active = %q", loaded.ActiveConfigVersion)
	}
	if _, ok := loaded.Config.Jobs.Get("intake"); !ok {
		t.Fatalf("fixture jobs missing: %v", loaded.Config.Jobs.Keys)
	}
}

func TestSeedDemoWipeRequiresDomainStore(t *testing.T) {
	_, err := SeedDemo(t.Context(), nil, NewMemoryConfigStore(), SeedOptions{WipeDomain: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSeedDemoRequiresConfigStore(t *testing.T) {
	_, err := SeedDemo(t.Context(), NewMemoryDomainStore(), nil, SeedOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFixtureConfigRolesMatchesTruckLoop(t *testing.T) {
	roles, err := FixtureConfigRoles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"administration", "logistic", "line"}
	if len(roles) != len(want) {
		t.Fatalf("roles = %#v, want %#v", roles, want)
	}
	for i := range want {
		if roles[i] != want[i] {
			t.Fatalf("roles = %#v, want %#v", roles, want)
		}
	}
}

func TestSeedDemoReplaceIsNotHarnessRunsAPI(t *testing.T) {
	// Guard: living reset is SeedDemo (Domain wipe + fixture), not /api/runs PUT.
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := domains.Get(t.Context(), "feragame"); err != nil && !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}
