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

func TestSeedDemoWipesDomainAndSeedsStagedFixture(t *testing.T) {
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
	if _, ok := got.Kind("load")["L1"]; ok {
		t.Fatal("pre-wipe L1 must not survive staged rebuild")
	}
	assertStagedDemoDomain(t, got)

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
}

func TestSeedDemoFillsMissingDomainWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false})
	if err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), cfg.Org)
	if err != nil {
		t.Fatal(err)
	}
	assertStagedDemoDomain(t, got)
}

func TestSeedDemoFillsEmptyDomainWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if err := domains.Put(t.Context(), "feragame", EmptyDomain()); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false})
	if err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), cfg.Org)
	if err != nil {
		t.Fatal(err)
	}
	assertStagedDemoDomain(t, got)
}

func TestSeedDemoSkipEmptyDomainStaging(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if err := domains.Put(t.Context(), "feragame", EmptyDomain()); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{SkipEmptyDomainStaging: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" {
		t.Fatalf("cfg = %#v", cfg)
	}
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.JobInstances) != 0 || len(got.Submissions) != 0 {
		t.Fatalf("empty Domain staged despite SkipEmptyDomainStaging: %#v", got)
	}
	version, err := configs.ActiveVersion(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if version != DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}
}

func TestSeedDemoStagedDomainSplitRoleWorklists(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: true}); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}

	admin, err := loader.Load(t.Context(), "feragame", OperatorIdentity{
		UserID: "administration", Org: "feragame", Roles: []string{"administration"},
	})
	if err != nil {
		t.Fatal(err)
	}
	logistic, err := loader.Load(t.Context(), "feragame", OperatorIdentity{
		UserID: "logistic", Org: "feragame", Roles: []string{"logistic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	line, err := loader.Load(t.Context(), "feragame", OperatorIdentity{
		UserID: "line", Org: "feragame", Roles: []string{"line"},
	})
	if err != nil {
		t.Fatal(err)
	}

	adminRefs := CollectActionableTaskRefs(admin.Config, admin.Worklist)
	logRefs := CollectActionableTaskRefs(logistic.Config, logistic.Worklist)
	lineRefs := CollectActionableTaskRefs(line.Config, line.Worklist)

	if !hasNestedActionableRef(adminRefs, "intake", "lots", SeedTruckLotsOpenID, "register") {
		t.Fatalf("administration actionable missing lots/register@%s: %#v", SeedTruckLotsOpenID, adminRefs)
	}
	if !hasActionableRef(adminRefs, "intake", "weightCheck", SeedTruckWeightCheckID) {
		t.Fatalf("administration actionable missing weightCheck@%s: %#v", SeedTruckWeightCheckID, adminRefs)
	}
	if hasNestedActionableRef(adminRefs, "line", "palletWeight", SeedLineWeighOpenID, "weigh") {
		t.Fatalf("administration must not see weigh: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "line", "lineLoad", SeedLineLineLoadID) {
		t.Fatalf("administration must not see lineLoad: %#v", adminRefs)
	}

	if !hasNestedActionableRef(logRefs, "line", "palletWeight", SeedLineWeighOpenID, "weigh") {
		t.Fatalf("logistic actionable missing weigh@%s: %#v", SeedLineWeighOpenID, logRefs)
	}
	if hasNestedActionableRef(logRefs, "intake", "lots", SeedTruckLotsOpenID, "register") {
		t.Fatalf("logistic must not see lots/register: %#v", logRefs)
	}
	if hasActionableRef(logRefs, "intake", "weightCheck", SeedTruckWeightCheckID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}
	if hasActionableRef(logRefs, "line", "lineLoad", SeedLineLineLoadID) {
		t.Fatalf("logistic must not see lineLoad: %#v", logRefs)
	}

	if !hasActionableRef(lineRefs, "line", "lineLoad", SeedLineLineLoadID) {
		t.Fatalf("line actionable missing lineLoad@%s: %#v", SeedLineLineLoadID, lineRefs)
	}
	if hasNestedActionableRef(lineRefs, "intake", "lots", SeedTruckLotsOpenID, "register") {
		t.Fatalf("line must not see lots/register: %#v", lineRefs)
	}
	if hasNestedActionableRef(lineRefs, "line", "palletWeight", SeedLineWeighOpenID, "weigh") {
		t.Fatalf("line must not see weigh: %#v", lineRefs)
	}

	adminIntake, _ := worklistJobByID(admin.Worklist, "intake")
	logIntake, _ := worklistJobByID(logistic.Worklist, "intake")
	lineJobWL, _ := worklistJobByID(line.Worklist, "line")
	if !adminIntake.CanStart {
		t.Fatal("administration canStart intake want true")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false")
	}
	if !lineJobWL.CanStart {
		t.Fatal("line canStart want true (no startsWhen gate on plant line)")
	}
}

func hasNestedActionableRef(refs []ActionableTaskRef, jobID, groupID, jobInstanceID, taskID string) bool {
	for _, r := range refs {
		if r.JobID == jobID && r.GroupID == groupID && r.JobInstanceID == jobInstanceID && r.TaskID == taskID {
			return true
		}
	}
	return false
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
	want := []string{"administration", "line", "logistic"}
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

func assertStagedDemoDomain(t *testing.T, got Domain) {
	t.Helper()
	if len(got.Kind("load")) < 1 {
		t.Fatalf("staged loads = %#v, want >= 1", got.Kind("load"))
	}
	if len(got.Kind("truck")) < 2 {
		t.Fatalf("staged trucks = %#v, want >= 2", got.Kind("truck"))
	}
	if len(got.JobInstances) < 4 {
		t.Fatalf("jobInstances = %d, want >= 4 (2 intake + 2 line)", len(got.JobInstances))
	}
	if len(got.Submissions) < 5 {
		t.Fatalf("submissions = %d, want >= 5", len(got.Submissions))
	}

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	wl := ProjectWorklist(cfg, got, FixtureAllRolesOp())
	intake, ok := worklistJobByID(wl, "intake")
	if !ok {
		t.Fatal("missing intake worklist")
	}
	lotsOpen, ok := worklistInstance(intake, SeedTruckLotsOpenID)
	if !ok || lotsOpen.Tasks["truckArrive"] != TaskStatusDone || lotsOpen.Tasks["lots"] != TaskStatusAvailable {
		t.Fatalf("lots-open truck = %#v", lotsOpen)
	}
	if g := lotsOpen.Groups["lots"]; !g.CanSpawn {
		t.Fatalf("lots-open chrome = %#v", g)
	}
	wc, ok := worklistInstance(intake, SeedTruckWeightCheckID)
	if !ok || wc.Tasks["lots"] != TaskStatusDone || wc.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck ready = %#v", wc)
	}
	line, ok := worklistJobByID(wl, "line")
	if !ok {
		t.Fatal("missing line worklist")
	}
	weighOpen, ok := worklistInstance(line, SeedLineWeighOpenID)
	if !ok || weighOpen.Tasks["openTurno"] != TaskStatusDone || weighOpen.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("weigh-open line = %#v", weighOpen)
	}
	lineLoad, ok := worklistInstance(line, SeedLineLineLoadID)
	if !ok || lineLoad.Tasks["palletWeight"] != TaskStatusDone || lineLoad.Tasks["lineLoad"] != TaskStatusActionable {
		t.Fatalf("lineLoad ready = %#v", lineLoad)
	}
}
