package org

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
	if err := domains.Put(t.Context(), FeragameScope(), seeded); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	version, err := configs.ActiveVersion(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if version != DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}

	got, err := domains.Get(t.Context(), FeragameScope())
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
	if err := domains.Put(t.Context(), FeragameScope(), seeded); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: true})
	if err != nil {
		t.Fatal(err)
	}

	got, err := domains.Get(t.Context(), mustScope(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Kind("lot")["L1"]; ok {
		t.Fatal("pre-wipe L1 must not survive staged rebuild")
	}
	assertStagedDemoDomain(t, got)

	loader := &DomainLoader{Domains: domains, Configs: configs}
	loaded, err := loader.Load(t.Context(), mustScope(t, cfg), FixtureAllRolesOp())
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
	passports := NewMemoryDPPStore()
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "other", SubjectID: "X", GTIN: "2", Lot: "b", Serial: "t",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/2/b/t",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		WipeDomain: true,
		DPPs:       passports,
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
	got, err := domains.Get(t.Context(), mustScope(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	assertStagedDemoDomain(t, got)
}

func TestSeedDemoFillsEmptyDomainWithoutWipe(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if err := domains.Put(t.Context(), FeragameScope(), EmptyDomain()); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false})
	if err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), mustScope(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	assertStagedDemoDomain(t, got)
}

func TestSeedDemoSkipEmptyDomainStaging(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if err := domains.Put(t.Context(), FeragameScope(), EmptyDomain()); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{SkipEmptyDomainStaging: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" {
		t.Fatalf("cfg = %#v", cfg)
	}
	got, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.JobInstances) != 0 || len(got.Submissions) != 0 {
		t.Fatalf("empty Domain staged despite SkipEmptyDomainStaging: %#v", got)
	}
	version, err := configs.ActiveVersion(t.Context(), FeragameScope())
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
	passports := NewMemoryDPPStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "L1"}}
	if err := domains.Put(t.Context(), FeragameScope(), seeded); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope: SeedScopeFeragameConfig,
		DPPs:  passports,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	version, err := configs.ActiveVersion(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if version != DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}
	got, err := domains.Get(t.Context(), FeragameScope())
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
	passports := NewMemoryDPPStore()
	seeded := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	seeded.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "L1"}}
	if err := domains.Put(t.Context(), FeragameScope(), seeded); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope:      SeedScopeFeragameConfig,
		DPPs:       passports,
		KeepDomain: true,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), FeragameScope())
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

func TestSeedDemoFeragameDomainWipesAndStages(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryDPPStore()
	marker := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	marker.Submissions = []Submission{{JobID: "intake", TaskID: "register"}}
	if err := domains.Put(t.Context(), FeragameScope(), marker); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope: SeedScopeFeragameDomain,
		DPPs:  passports,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	got, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Kind("lot")["L1"]; ok {
		t.Fatal("pre-wipe L1 must not survive feragame-domain rebuild")
	}
	assertStagedDemoDomain(t, got)
}

func TestSeedDemoFeragameFullWipesAndStages(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryDPPStore()
	marker := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	marker.Submissions = []Submission{{JobID: "intake", TaskID: "register"}}
	if err := domains.Put(t.Context(), FeragameScope(), marker); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "feragame", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}

	cfg, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope: SeedScopeFeragameFull,
		DPPs:  passports,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "feragame" || cfg.Version != DefaultFixtureVersion {
		t.Fatalf("cfg = %#v", cfg)
	}
	got, err := domains.Get(t.Context(), FeragameScope())
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
	loader := &DomainLoader{Domains: domains, Configs: configs}

	admin, err := loader.Load(t.Context(), FeragameScope(), OperatorIdentity{
		UserID: "administration", Org: "feragame", Roles: []string{"administration"},
	})
	if err != nil {
		t.Fatal(err)
	}
	logistic, err := loader.Load(t.Context(), FeragameScope(), OperatorIdentity{
		UserID: "logistic", Org: "feragame", Roles: []string{"logistic"},
	})
	if err != nil {
		t.Fatal(err)
	}
	line, err := loader.Load(t.Context(), FeragameScope(), OperatorIdentity{
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
	for _, lotID := range []string{SeedLoadCOID, SeedLoadDAID} {
		if hasActionableRef(adminRefs, "lot_processing", "weigh", lotID) {
			t.Fatalf("administration must not see weigh: %#v", adminRefs)
		}
		if hasActionableRef(adminRefs, "lot_processing", "markProgress", lotID) {
			t.Fatalf("administration must not see markProgress: %#v", adminRefs)
		}
		if hasActionableRef(adminRefs, "lot_processing", "containerReg", lotID) {
			t.Fatalf("administration must not see containerReg: %#v", adminRefs)
		}
		if hasActionableRef(adminRefs, "lot_processing", "closeLot", lotID) {
			t.Fatalf("administration must not see closeLot: %#v", adminRefs)
		}
	}

	// Staged seed closed both lots; logistic peer Tasks are unavailable (no open lot).
	for _, lotID := range []string{SeedLoadCOID, SeedLoadDAID} {
		if hasActionableRef(logRefs, "lot_processing", "containerReg", lotID) {
			t.Fatalf("logistic must not see containerReg after closeLot: %#v", logRefs)
		}
		if hasActionableRef(logRefs, "lot_processing", "weigh", lotID) {
			t.Fatalf("logistic must not see weigh: %#v", logRefs)
		}
		if hasActionableRef(lineRefs, "lot_processing", "markProgress", lotID) {
			t.Fatalf("line must not see markProgress after staged closeLot: %#v", lineRefs)
		}
		if hasActionableRef(lineRefs, "lot_processing", "weigh", lotID) {
			t.Fatalf("line must not see weigh after staged closeLot: %#v", lineRefs)
		}
		if hasActionableRef(lineRefs, "lot_processing", "containerReg", lotID) {
			t.Fatalf("line must not see containerReg: %#v", lineRefs)
		}
		if hasActionableRef(lineRefs, "lot_processing", "closeLot", lotID) {
			t.Fatalf("line must not see closeLot: %#v", lineRefs)
		}
	}
	if hasActionableRef(logRefs, "intake", "weightCheck", SeedTruckID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}

	for _, lotID := range []string{SeedLoadCOID, SeedLoadDAID} {
		if !hasActionableRef(adminRefs, "lot_processing", "lock", lotID) {
			t.Fatalf("administration must see lot_processing/lock on %s: %#v", lotID, adminRefs)
		}
		if hasActionableRef(logRefs, "lot_processing", "lock", lotID) {
			t.Fatalf("logistic must not see lock: %#v", logRefs)
		}
		if hasActionableRef(lineRefs, "lot_processing", "lock", lotID) {
			t.Fatalf("line must not see lock: %#v", lineRefs)
		}
	}
	if _, ok := worklistJobByID(admin.Worklist, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}

	adminIntake, _ := worklistJobByID(admin.Worklist, "intake")
	logIntake, _ := worklistJobByID(logistic.Worklist, "intake")
	lineLot, _ := worklistJobByID(line.Worklist, "lot_processing")
	logLot, _ := worklistJobByID(logistic.Worklist, "lot_processing")
	adminLot, _ := worklistJobByID(admin.Worklist, "lot_processing")
	// Seed truck is mid-intake; parallel trucks keep Nuovo camion available for administration.
	if !adminIntake.CanStart {
		t.Fatal("administration canStart intake want true while prior truck incomplete (parallel)")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false")
	}
	// Both lots fullyProcessed — no eligible startLot; line/admin lack startLot role.
	if logLot.CanStart {
		t.Fatal("logistic canStart lot_processing want false after both lots closed")
	}
	if lineLot.CanStart {
		t.Fatal("line canStart lot_processing want false")
	}
	if adminLot.CanStart {
		t.Fatal("administration canStart lot_processing want false")
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
	// Guard: org reset is SeedDemo (Domain wipe + fixture), not /api/runs PUT.
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := domains.Get(t.Context(), FeragameScope()); err != nil && !errors.Is(err, ErrNotFound) {
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
	if _, ok := got.Kind("truck")[SeedTruckID]; !ok {
		t.Fatalf("staged truck missing %s: %#v", SeedTruckID, got.Kind("truck"))
	}
	if len(got.Kind("pallet")) != 4 {
		t.Fatalf("staged pallets = %#v, want 4", got.Kind("pallet"))
	}
	if len(got.Kind("workingDay")) != 0 {
		t.Fatalf("staged workingDay must be absent: %#v", got.Kind("workingDay"))
	}
	if len(got.JobInstances) != 3 {
		t.Fatalf("jobInstances = %d, want 3 (1 intake + 2 lot_processing)", len(got.JobInstances))
	}
	// truckArrive + 2× register + 2× startLot + 4× weigh + 2× markProgress + 2× containerReg + 2× closeLot
	if len(got.Submissions) < 15 {
		t.Fatalf("submissions = %d, want >= 15", len(got.Submissions))
	}
	for _, sub := range got.Submissions {
		switch sub.TaskID {
		case "open", "openTurno", "closeTurno":
			t.Fatalf("staged demo must not Commit day/turno Task %s/%s", sub.JobID, sub.TaskID)
		}
		if sub.JobID == "workingDay" || sub.JobID == "line" {
			t.Fatalf("staged demo must not Commit obsolete Job %s", sub.JobID)
		}
	}

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	wl := projectWorklistUnfiltered(cfg, got, FixtureAllRolesOp())
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
	lotProc, ok := worklistJobByID(wl, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing worklist")
	}
	if lotProc.CanStart {
		t.Fatal("lot_processing CanStart want false after both lots fullyProcessed")
	}
	for _, lotID := range []string{SeedLoadCOID, SeedLoadDAID} {
		inst, ok := worklistInstance(lotProc, lotID)
		if !ok || inst.Tasks["startLot"] != TaskStatusDone {
			t.Fatalf("lot_processing %s = %#v", lotID, inst)
		}
		for _, id := range []string{"weigh", "markProgress", "containerReg", "closeLot"} {
			if inst.Tasks[id] != TaskStatusUnavailable {
				t.Fatalf("%s@%s = %q, want unavailable after closeLot", id, lotID, inst.Tasks[id])
			}
		}
		if _, ok := inst.Tasks["openTurno"]; ok {
			t.Fatalf("lot_processing must not expose openTurno: %#v", inst.Tasks)
		}
		if _, ok := inst.Tasks["closeTurno"]; ok {
			t.Fatalf("lot_processing must not expose closeTurno: %#v", inst.Tasks)
		}
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
	if _, ok := worklistJobByID(wl, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}
	refs := CollectActionableTaskRefs(cfg, wl)
	for _, loadID := range []string{SeedLoadCOID, SeedLoadDAID} {
		if !hasActionableRef(refs, "lot_processing", "lock", loadID) {
			t.Fatalf("lot_processing/lock must be actionable on %s: %#v", loadID, refs)
		}
	}
}
