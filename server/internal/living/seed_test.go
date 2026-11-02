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

	adminRefs := CollectAvailableTaskRefs(admin.Config, admin.Worklist, []string{"administration"})
	logRefs := CollectAvailableTaskRefs(logistic.Config, logistic.Worklist, []string{"logistic"})
	lineRefs := CollectAvailableTaskRefs(line.Config, line.Worklist, []string{"line"})

	if !hasAvailableRef(adminRefs, "intake", "codiceFir", SeedIntakeAfterRegisterID) {
		t.Fatalf("administration Available missing codiceFir@%s: %#v", SeedIntakeAfterRegisterID, adminRefs)
	}
	if !hasAvailableRef(adminRefs, "intake", "weightCheck", SeedIntakeWeightCheckID) {
		t.Fatalf("administration Available missing weightCheck@%s: %#v", SeedIntakeWeightCheckID, adminRefs)
	}
	if hasAvailableRef(adminRefs, "intake", "palletWeight", SeedIntakeAfterFirID) {
		t.Fatalf("administration must not see palletWeight: %#v", adminRefs)
	}
	if hasAvailableRef(adminRefs, "line", "lineLoad", SeedLineShiftID) {
		t.Fatalf("administration must not see lineLoad: %#v", adminRefs)
	}

	if !hasAvailableRef(logRefs, "intake", "palletWeight", SeedIntakeAfterFirID) {
		t.Fatalf("logistic Available missing palletWeight@%s: %#v", SeedIntakeAfterFirID, logRefs)
	}
	if hasAvailableRef(logRefs, "intake", "codiceFir", SeedIntakeAfterRegisterID) {
		t.Fatalf("logistic must not see codiceFir: %#v", logRefs)
	}
	if hasAvailableRef(logRefs, "intake", "weightCheck", SeedIntakeWeightCheckID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}
	if hasAvailableRef(logRefs, "line", "lineLoad", SeedLineShiftID) {
		t.Fatalf("logistic must not see lineLoad: %#v", logRefs)
	}

	if !hasAvailableRef(lineRefs, "line", "lineLoad", SeedLineShiftID) {
		t.Fatalf("line Available missing lineLoad@%s: %#v", SeedLineShiftID, lineRefs)
	}
	if hasAvailableRef(lineRefs, "intake", "codiceFir", SeedIntakeAfterRegisterID) {
		t.Fatalf("line must not see intake codiceFir: %#v", lineRefs)
	}
	if hasAvailableRef(lineRefs, "intake", "palletWeight", SeedIntakeAfterFirID) {
		t.Fatalf("line must not see palletWeight: %#v", lineRefs)
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
		t.Fatal("line canStart want true after palletWeight gate")
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

func assertStagedDemoDomain(t *testing.T, got Domain) {
	t.Helper()
	if len(got.Kind("load")) < 3 {
		t.Fatalf("staged loads = %#v, want >= 3", got.Kind("load"))
	}
	if len(got.JobInstances) < 4 {
		t.Fatalf("jobInstances = %d, want >= 4 (3 intake + line)", len(got.JobInstances))
	}
	if len(got.Submissions) < 6 {
		t.Fatalf("submissions = %d, want >= 6", len(got.Submissions))
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
	reg, ok := worklistInstance(intake, SeedIntakeAfterRegisterID)
	if !ok || reg.Tasks["register"] != TaskStatusDone || reg.Tasks["codiceFir"] != TaskStatusAvailable {
		t.Fatalf("after register = %#v", reg)
	}
	fir, ok := worklistInstance(intake, SeedIntakeAfterFirID)
	if !ok || fir.Tasks["codiceFir"] != TaskStatusDone || fir.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("after codiceFir / palletWeight available = %#v", fir)
	}
	wc, ok := worklistInstance(intake, SeedIntakeWeightCheckID)
	if !ok || wc.Tasks["palletWeight"] != TaskStatusDone || wc.Tasks["weightCheck"] != TaskStatusAvailable {
		t.Fatalf("weightCheck ready = %#v", wc)
	}
	line, ok := worklistJobByID(wl, "line")
	if !ok {
		t.Fatal("missing line worklist")
	}
	shift, ok := worklistInstance(line, SeedLineShiftID)
	if !ok || shift.Tasks["shift"] != TaskStatusDone || shift.Tasks["lineLoad"] != TaskStatusAvailable {
		t.Fatalf("line after shift = %#v", shift)
	}
}
