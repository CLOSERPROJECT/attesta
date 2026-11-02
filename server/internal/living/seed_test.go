package living

import (
	"errors"
	"testing"
)

func TestSeedDemoSetsActiveConfigWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	seeded := domainFromKinds(map[string]EntityMap{"load": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{
		{JobID: "intake", TaskID: "register", JobInstanceID: "L1", UserID: "ada"},
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
	if len(got.Kind("load")) != 1 || len(got.Submissions) != 1 {
		t.Fatalf("domain wiped without WipeDomain: %#v", got)
	}
}

func TestSeedDemoWipesDomainAndSeedsFixture(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	seeded := domainFromKinds(map[string]EntityMap{"load": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{{JobID: "intake", TaskID: "register"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
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
	if _, ok := got.Kind("load")[DemoIntakeInstanceID]; !ok {
		t.Fatalf("demo intake load missing: %#v", got.Kind("load"))
	}
	if len(got.Kind("pallet")) != 2 {
		t.Fatalf("demo pallets = %#v", got.Kind("pallet"))
	}
	if len(got.Submissions) != 4 {
		t.Fatalf("submissions = %#v", got.Submissions)
	}
	last := got.Submissions[len(got.Submissions)-1]
	if last.JobID != "line" || last.TaskID != "shift" || last.JobInstanceID != DemoLineShiftID {
		t.Fatalf("last Submission = %#v", last)
	}
	foundLine := false
	for _, inst := range got.JobInstances {
		if inst.JobID == "line" && inst.ID == DemoLineShiftID {
			foundLine = true
			break
		}
	}
	if !foundLine {
		t.Fatalf("line Job instance missing: %#v", got.JobInstances)
	}

	loader := &Loader{Domains: domains, Configs: configs}
	loaded, err := loader.Load(t.Context(), cfg.Org, FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActiveConfigVersion != DefaultFixtureVersion {
		t.Fatalf("active = %q", loaded.ActiveConfigVersion)
	}
	if _, ok := loaded.Config.Jobs.Get("intake"); !ok {
		t.Fatalf("fixture jobs missing: %v", loaded.Config.Jobs.Keys)
	}

	line, ok := worklistJobByID(loaded.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist job")
	}
	inst, ok := worklistInstance(line, DemoLineShiftID)
	if !ok {
		t.Fatalf("missing line instance %q: %#v", DemoLineShiftID, line.Instances)
	}
	if inst.Tasks["lineLoad"] != TaskStatusAvailable {
		t.Fatalf("lineLoad = %q, want available", inst.Tasks["lineLoad"])
	}
	lineLoad := findBundle(loaded.TaskFormBundles, "line", "lineLoad", DemoLineShiftID)
	if lineLoad == nil {
		t.Fatalf("missing lineLoad bundle; got %#v", addresses(loaded.TaskFormBundles))
	}
	if len(lineLoad.RowIDs) == 0 {
		t.Fatal("lineLoad bundle needs weighed pallet row ids")
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

func TestFixtureConfigRolesMatchesFeragameConfig(t *testing.T) {
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
