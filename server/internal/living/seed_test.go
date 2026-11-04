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

	if !hasActionableRef(adminRefs, "intake", "weightCheck", SeedTruckID) {
		t.Fatalf("administration actionable missing weightCheck@%s: %#v", SeedTruckID, adminRefs)
	}
	if hasNestedActionableRef(adminRefs, "line", "palletWeight", SeedLineID, "weigh") {
		t.Fatalf("administration must not see weigh: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "line", "lineLoad", SeedLineID) {
		t.Fatalf("administration must not see lineLoad: %#v", adminRefs)
	}
	if hasNestedActionableRef(adminRefs, "line", "containerReg", SeedLineID, "register") {
		t.Fatalf("administration must not see register: %#v", adminRefs)
	}

	if !hasNestedActionableRef(logRefs, "line", "containerReg", SeedLineID, "register") {
		t.Fatalf("logistic actionable missing register@%s: %#v", SeedLineID, logRefs)
	}
	if hasActionableRef(logRefs, "intake", "weightCheck", SeedTruckID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}
	if hasActionableRef(logRefs, "line", "lineLoad", SeedLineID) {
		t.Fatalf("logistic must not see lineLoad: %#v", logRefs)
	}

	if hasActionableRef(lineRefs, "line", "lineLoad", SeedLineID) {
		t.Fatalf("line must not see lineLoad after staged close: %#v", lineRefs)
	}
	if hasNestedActionableRef(lineRefs, "line", "containerReg", SeedLineID, "register") {
		t.Fatalf("line must not see register: %#v", lineRefs)
	}
	if hasNestedActionableRef(lineRefs, "line", "palletWeight", SeedLineID, "weigh") {
		t.Fatalf("line must not see weigh: %#v", lineRefs)
	}
	adminPassport, _ := worklistJobByID(admin.Worklist, "passport")
	if !adminPassport.CanStart || adminPassport.StartLabel != "Emetti DPP" {
		t.Fatalf("administration passport CanStart = %#v, want true + Emetti DPP", adminPassport)
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
	loads := got.Kind("load")
	if loads[SeedLoadNOID] == nil || loads[SeedLoadSUID] == nil {
		t.Fatalf("staged loads = %#v, want %s and %s", loads, SeedLoadNOID, SeedLoadSUID)
	}
	if len(got.Kind("truck")) != 1 {
		t.Fatalf("staged trucks = %#v, want 1", got.Kind("truck"))
	}
	if len(got.Kind("pallet")) != 4 {
		t.Fatalf("staged pallets = %#v, want 4", got.Kind("pallet"))
	}
	if len(got.JobInstances) != 2 {
		t.Fatalf("jobInstances = %d, want 2 (1 intake + 1 line)", len(got.JobInstances))
	}
	// truckArrive + 2× lots/register + openTurno + 4× weigh
	if len(got.Submissions) < 8 {
		t.Fatalf("submissions = %d, want >= 8", len(got.Submissions))
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
	truck, ok := worklistInstance(intake, SeedTruckID)
	if !ok || truck.Tasks["truckArrive"] != TaskStatusDone || truck.Tasks["lots"] != TaskStatusDone {
		t.Fatalf("intake truck = %#v", truck)
	}
	if truck.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck = %q, want actionable", truck.Tasks["weightCheck"])
	}
	line, ok := worklistJobByID(wl, "line")
	if !ok {
		t.Fatal("missing line worklist")
	}
	shift, ok := worklistInstance(line, SeedLineID)
	if !ok || shift.Tasks["openTurno"] != TaskStatusDone || shift.Tasks["palletWeight"] != TaskStatusDone {
		t.Fatalf("line shift = %#v", shift)
	}
	if shift.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad = %q, want done", shift.Tasks["lineLoad"])
	}
	for _, loadID := range []string{SeedLoadNOID, SeedLoadSUID} {
		facts := loads[loadID]
		if facts["fullyProcessed"] != true {
			t.Fatalf("load %s fullyProcessed = %#v, want true", loadID, facts["fullyProcessed"])
		}
		if facts["locked"] == true {
			t.Fatalf("load %s must not be passport-locked", loadID)
		}
	}
	passport, ok := worklistJobByID(wl, "passport")
	if !ok || !passport.CanStart {
		t.Fatalf("passport CanStart = %#v, want true with two eligible loads", passport)
	}
}
