package living

import (
	"errors"
	"strings"
	"testing"
)

func TestSeedDemoSetsActiveConfigWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
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
	if len(got.Kind("lot")) != 1 || len(got.Submissions) != 1 {
		t.Fatalf("domain wiped without WipeDomain: %#v", got)
	}
}

func TestSeedDemoWipesDomainAndSeedsStagedFixture(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
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
	if _, ok := got.Kind("lot")["L1"]; ok {
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

func TestSeedDemoWipeClearsPassports(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryPassportStore()
	if err := passports.Create(t.Context(), PassportRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), PassportRecord{
		Org: "other", SubjectID: "X", GTIN: "2", Lot: "b", Serial: "t",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/2/b/t",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		WipeDomain: true,
		Passports:  passports,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := passports.Get(t.Context(), "feragame", SeedLoadCOID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("feragame passport after wipe = %v, want ErrNotFound", err)
	}
	if _, err := passports.Get(t.Context(), "other", "X"); err != nil {
		t.Fatalf("other org passport must survive: %v", err)
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

func TestSeedDemoFeragameConfigWipesToEmptyWithoutStaging(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryPassportStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "L1"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), PassportRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope:     SeedScopeFeragameConfig,
		Passports: passports,
	})
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
	if len(got.JobInstances) != 0 || len(got.Submissions) != 0 || len(got.Kind("lot")) != 0 {
		t.Fatalf("feragame-config Domain not empty: %#v", got)
	}
	if _, ok := got.Kind("truck")[SeedTruckID]; ok {
		t.Fatal("feragame-config must not stage Domain")
	}
	if _, err := passports.Get(t.Context(), "feragame", SeedLoadCOID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("passport after feragame-config = %v, want ErrNotFound", err)
	}
}

func TestSeedDemoFeragameConfigRequiresDomainStore(t *testing.T) {
	_, err := SeedDemo(t.Context(), nil, NewMemoryConfigStore(), SeedOptions{Scope: SeedScopeFeragameConfig})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "domain store required") {
		t.Fatalf("err = %v, want domain store required", err)
	}
}

func TestSeedDemoFeragameConfigKeepDomainLeavesDomain(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryPassportStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "L1"}}
	if err := domains.Put(t.Context(), "feragame", seeded); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), PassportRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope:      SeedScopeFeragameConfig,
		Passports:  passports,
		KeepDomain: true,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["L1"] == nil || len(got.Submissions) != 1 {
		t.Fatalf("Domain must be kept: %#v", got)
	}
	if _, err := passports.Get(t.Context(), "feragame", SeedLoadCOID); err != nil {
		t.Fatalf("passport must be kept: %v", err)
	}
}

func TestSeedDemoKeepDomainRejectedForDomainScope(t *testing.T) {
	_, err := SeedDemo(t.Context(), NewMemoryDomainStore(), NewMemoryConfigStore(), SeedOptions{
		Scope:      SeedScopeFeragameDomain,
		KeepDomain: true,
	})
	if err == nil || !strings.Contains(err.Error(), "KeepDomain") {
		t.Fatalf("err = %v, want KeepDomain rejection", err)
	}
}

func TestSeedDemoFeragameFullWipesAndStages(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryPassportStore()
	marker := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	marker.Submissions = []Submission{{JobID: "intake", TaskID: "register"}}
	if err := domains.Put(t.Context(), "feragame", marker); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), PassportRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope:     SeedScopeFeragameFull,
		Passports: passports,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Kind("lot")["L1"]; ok {
		t.Fatal("pre-wipe L1 must not survive feragame-full rebuild")
	}
	assertStagedDemoDomain(t, got)
	if _, err := passports.Get(t.Context(), "feragame", SeedLoadCOID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("passport after feragame-full = %v, want ErrNotFound", err)
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
	if hasActionableRef(adminRefs, "line", "weigh", SeedLineID) {
		t.Fatalf("administration must not see weigh: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "line", "markProgress", SeedLineID) {
		t.Fatalf("administration must not see markProgress: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "line", "containerReg", SeedLineID) {
		t.Fatalf("administration must not see containerReg: %#v", adminRefs)
	}

	// Staged seed closed both lots; logistic peer Tasks are unavailable (no open lot).
	if hasActionableRef(logRefs, "line", "containerReg", SeedLineID) {
		t.Fatalf("logistic must not see containerReg after closeLot: %#v", logRefs)
	}
	if hasActionableRef(logRefs, "intake", "weightCheck", SeedTruckID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}
	if hasActionableRef(logRefs, "line", "weigh", SeedLineID) {
		t.Fatalf("logistic must not see weigh after closeLot: %#v", logRefs)
	}

	if hasActionableRef(lineRefs, "line", "markProgress", SeedLineID) {
		t.Fatalf("line must not see markProgress after staged closeLot: %#v", lineRefs)
	}
	if hasActionableRef(lineRefs, "line", "containerReg", SeedLineID) {
		t.Fatalf("line must not see containerReg: %#v", lineRefs)
	}
	if hasActionableRef(lineRefs, "line", "weigh", SeedLineID) {
		t.Fatalf("line must not see weigh: %#v", lineRefs)
	}
	if !hasActionableRef(lineRefs, "line", "closeTurno", SeedLineID) {
		t.Fatalf("line actionable missing closeTurno@%s: %#v", SeedLineID, lineRefs)
	}
	if !hasActionableRef(logRefs, "line", "closeTurno", SeedLineID) {
		t.Fatalf("logistic actionable missing closeTurno@%s: %#v", SeedLineID, logRefs)
	}
	adminPassport, _ := worklistJobByID(admin.Worklist, "passport")
	if !adminPassport.CanStart || adminPassport.StartLabel != "Emetti DPP" {
		t.Fatalf("administration passport CanStart = %#v, want true + Emetti DPP", adminPassport)
	}

	adminIntake, _ := worklistJobByID(admin.Worklist, "intake")
	logIntake, _ := worklistJobByID(logistic.Worklist, "intake")
	lineJobWL, _ := worklistJobByID(line.Worklist, "line")
	logLine, _ := worklistJobByID(logistic.Worklist, "line")
	// Seed truck is mid-intake (weightCheck still open) — Nuovo camion stays off.
	if adminIntake.CanStart {
		t.Fatal("administration canStart intake want false while prior truck incomplete")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false")
	}
	if lineJobWL.CanStart {
		t.Fatal("line canStart want false while seed shift is still open")
	}
	if logLine.CanStart {
		t.Fatal("logistic canStart line want false while seed shift is still open")
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
	loads := got.Kind("lot")
	if loads[SeedLoadCOID] == nil || loads[SeedLoadDAID] == nil {
		t.Fatalf("staged loads = %#v, want %s and %s", loads, SeedLoadCOID, SeedLoadDAID)
	}
	if len(got.Kind("truck")) != 1 {
		t.Fatalf("staged trucks = %#v, want 1", got.Kind("truck"))
	}
	if len(got.Kind("pallet")) != 4 {
		t.Fatalf("staged pallets = %#v, want 4", got.Kind("pallet"))
	}
	if len(got.JobInstances) != 3 {
		t.Fatalf("jobInstances = %d, want 3 (1 workingDay + 1 intake + 1 line)", len(got.JobInstances))
	}
	if _, ok := got.Kind("workingDay")["2026-11-02"]; !ok {
		t.Fatalf("staged workingDay missing: %#v", got.Kind("workingDay"))
	}
	// workingDay/open + truckArrive + 2× register + openTurno + 4× weigh + …
	if len(got.Submissions) < 9 {
		t.Fatalf("submissions = %d, want >= 9", len(got.Submissions))
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
	if !ok || truck.Tasks["truckArrive"] != TaskStatusDone || truck.Tasks["register"] != TaskStatusActionable {
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
	if !ok || shift.Tasks["openTurno"] != TaskStatusDone {
		t.Fatalf("line shift = %#v", shift)
	}
	for _, id := range []string{"weigh", "markProgress", "containerReg", "closeLot", "startLot"} {
		if shift.Tasks[id] != TaskStatusUnavailable {
			t.Fatalf("%s = %q, want unavailable after both lots closeLot", id, shift.Tasks[id])
		}
	}
	if shift.Tasks["closeTurno"] != TaskStatusActionable {
		t.Fatalf("closeTurno = %q, want actionable", shift.Tasks["closeTurno"])
	}
	for _, loadID := range []string{SeedLoadCOID, SeedLoadDAID} {
		facts := loads[loadID]
		if facts["status"] != "fullyProcessed" {
			t.Fatalf("lot %s status = %#v, want fullyProcessed", loadID, facts["status"])
		}
		if facts["locked"] == true {
			t.Fatalf("lot %s must not be passport-locked", loadID)
		}
	}
	passport, ok := worklistJobByID(wl, "passport")
	if !ok || !passport.CanStart {
		t.Fatalf("passport CanStart = %#v, want true with two eligible loads", passport)
	}
}
