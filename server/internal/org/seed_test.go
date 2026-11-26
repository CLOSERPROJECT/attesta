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

func TestSeedDemoPrunesRetiredStreams(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	orphanScope := Scope{Org: "feragame", Stream: "recycling"}
	fixture := FeragameScope()

	orphanCfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), orphanScope, "1", orphanCfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), orphanScope, "1"); err != nil {
		t.Fatal(err)
	}
	orphanDomain := domainFromKinds(map[string]EntityMap{"lot": {"OLD": {"client": "X"}}})
	orphanDomain.Submissions = []Submission{{JobID: "intake", TaskID: "register"}}
	if err := domains.Put(t.Context(), orphanScope, orphanDomain); err != nil {
		t.Fatal(err)
	}
	keep := domainFromKinds(map[string]EntityMap{"lot": {"KEEP": {"client": "Y"}}})
	keep.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "KEEP"}}
	if err := domains.Put(t.Context(), fixture, keep); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{WipeDomain: false}); err != nil {
		t.Fatal(err)
	}

	streams, err := configs.ListStreams(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0] != fixture.Stream {
		t.Fatalf("active streams = %v, want [%q]", streams, fixture.Stream)
	}
	if _, err := configs.ActiveVersion(t.Context(), orphanScope); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan active = %v, want ErrNotFound", err)
	}
	if _, err := configs.Get(t.Context(), orphanScope, "1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan config = %v, want ErrNotFound", err)
	}
	if _, err := domains.Get(t.Context(), orphanScope); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan domain = %v, want ErrNotFound", err)
	}
	got, err := domains.Get(t.Context(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["KEEP"] == nil {
		t.Fatalf("fixture Domain must be kept without WipeDomain: %#v", got)
	}
}

func TestSeedDemoKeepDomainPrunesRetiredStreams(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	orphanScope := Scope{Org: "feragame", Stream: "recycling"}
	fixture := FeragameScope()

	orphanCfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), orphanScope, "1", orphanCfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), orphanScope, "1"); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), orphanScope, domainFromKinds(map[string]EntityMap{"lot": {"OLD": {}}})); err != nil {
		t.Fatal(err)
	}
	keep := domainFromKinds(map[string]EntityMap{"lot": {"L1": {"client": "NO"}}})
	keep.Submissions = []Submission{{JobID: "intake", TaskID: "register", JobInstanceID: "L1"}}
	if err := domains.Put(t.Context(), fixture, keep); err != nil {
		t.Fatal(err)
	}

	if _, err := SeedDemo(t.Context(), domains, configs, SeedOptions{
		Scope:      SeedScopeFeragameConfig,
		KeepDomain: true,
	}); err != nil {
		t.Fatal(err)
	}

	streams, err := configs.ListStreams(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0] != fixture.Stream {
		t.Fatalf("active streams = %v, want [%q]", streams, fixture.Stream)
	}
	if _, err := domains.Get(t.Context(), orphanScope); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan domain survived keep-domain: %v", err)
	}
	got, err := domains.Get(t.Context(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("lot")["L1"] == nil || len(got.Submissions) != 1 {
		t.Fatalf("fixture Domain must be kept: %#v", got)
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
		Org: "feragame", Stream: "pv-recovery", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/1/a/s",
	}); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: "other", Stream: "pv-recovery", SubjectID: "X", GTIN: "2", Lot: "b", Serial: "t",
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
	if _, err := passports.Get(t.Context(), FeragameScope(), SeedLoadCOID); err != nil {
		t.Fatalf("finished lot passport after wipe+stage = %v, want issued DPP", err)
	}
	if _, err := passports.Get(t.Context(), Scope{Org: "other", Stream: "pv-recovery"}, "X"); err != nil {
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
		Org: "feragame", Stream: "pv-recovery", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
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
	if _, err := passports.Get(t.Context(), FeragameScope(), SeedLoadCOID); !errors.Is(err, ErrNotFound) {
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
		Org: "feragame", Stream: "pv-recovery", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
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
	if _, err := passports.Get(t.Context(), FeragameScope(), SeedLoadCOID); err != nil {
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
		Org: "feragame", Stream: "pv-recovery", SubjectID: SeedLoadCOID, GTIN: "1", Lot: "a", Serial: "s",
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
	if _, err := passports.Get(t.Context(), FeragameScope(), SeedLoadCOID); err != nil {
		t.Fatalf("finished lot passport after feragame-full = %v, want issued DPP", err)
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

	if !hasActionableRef(adminRefs, "intake", "weightCheck", SeedTruckBID) {
		t.Fatalf("administration actionable missing weightCheck@%s: %#v", SeedTruckBID, adminRefs)
	}
	if hasActionableRef(adminRefs, "intake", "weightCheck", SeedTruckID) {
		t.Fatalf("truck A weightCheck must be done, not actionable: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "lot_processing", "weigh", SeedLoadDAID) {
		t.Fatalf("administration must not see weigh: %#v", adminRefs)
	}
	if hasActionableRef(adminRefs, "lot_processing", "lock", SeedLoadCOID) {
		t.Fatalf("finished lot lock must not stay actionable: %#v", adminRefs)
	}
	if hasActionableRef(logRefs, "intake", "weightCheck", SeedTruckBID) {
		t.Fatalf("logistic must not see weightCheck: %#v", logRefs)
	}
	if !hasActionableRef(lineRefs, "lot_processing", "weigh", SeedLoadDAID) {
		t.Fatalf("line must see weigh on open lot %s: %#v", SeedLoadDAID, lineRefs)
	}
	if hasActionableRef(lineRefs, "lot_processing", "weigh", SeedLoadCOID) {
		t.Fatalf("line must not see weigh on finished lot: %#v", lineRefs)
	}
	if _, ok := worklistJobByID(admin.Worklist, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}

	adminIntake, _ := worklistJobByID(admin.Worklist, "intake")
	logIntake, _ := worklistJobByID(logistic.Worklist, "intake")
	lineLot, _ := worklistJobByID(line.Worklist, "lot_processing")
	logLot, _ := worklistJobByID(logistic.Worklist, "lot_processing")
	adminLot, _ := worklistJobByID(admin.Worklist, "lot_processing")
	if !adminIntake.CanStart {
		t.Fatal("administration canStart intake want true (parallel trucks)")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false")
	}
	// Unfiltered Play: $stream.id unbound ⇒ startLot readyWhen false (no truck-scoped eligible lots).
	if logLot.CanStart {
		t.Fatal("logistic canStart lot_processing want false on unfiltered Play while DA is open")
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
	if loads[SeedLoadCOID] == nil || loads[SeedLoadDAID] == nil || loads[SeedTruckBLotID] == nil {
		t.Fatalf("staged loads = %#v, want %s, %s, and %s", loads, SeedLoadCOID, SeedLoadDAID, SeedTruckBLotID)
	}
	trucks := got.Kind("truck")
	if len(trucks) != 2 {
		t.Fatalf("staged trucks = %#v, want 2", trucks)
	}
	if _, ok := trucks[SeedTruckID]; !ok {
		t.Fatalf("staged truck A missing %s: %#v", SeedTruckID, trucks)
	}
	if _, ok := trucks[SeedTruckBID]; !ok {
		t.Fatalf("staged truck B missing %s: %#v", SeedTruckBID, trucks)
	}
	// Truck A: 2 lots × 2 pallets; truck B: 1 lot × 2 pallets.
	if len(got.Kind("pallet")) != 6 {
		t.Fatalf("staged pallets = %#v, want 6", got.Kind("pallet"))
	}
	if len(got.Kind("workingDay")) != 0 {
		t.Fatalf("staged workingDay must be absent: %#v", got.Kind("workingDay"))
	}
	// 2 intake + 2 lot_processing (CO finished, DA open); truck B has no lot_processing yet.
	if len(got.JobInstances) != 4 {
		t.Fatalf("jobInstances = %d, want 4 (2 intake + 2 lot_processing)", len(got.JobInstances))
	}
	// truckA: arrive+2×register+weightCheck+CO(full incl lock)+DA(start+1 weigh); truckB: arrive+register
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
	op := FixtureAllRolesOp()
	wl := projectWorklistUnfiltered(cfg, got, op)
	intake, ok := worklistJobByID(wl, "intake")
	if !ok {
		t.Fatal("missing intake worklist")
	}
	truckA, ok := worklistInstance(intake, SeedTruckID)
	if !ok || truckA.Tasks["truckArrive"] != TaskStatusDone || !truckA.Finished {
		t.Fatalf("truck A intake = %#v, want finished after weightCheck", truckA)
	}
	if truckA.Tasks["weightCheck"] != TaskStatusDone {
		t.Fatalf("truck A weightCheck = %q, want done", truckA.Tasks["weightCheck"])
	}
	truckB, ok := worklistInstance(intake, SeedTruckBID)
	if !ok || truckB.Tasks["truckArrive"] != TaskStatusDone {
		t.Fatalf("truck B intake = %#v", truckB)
	}
	if truckB.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("truck B weightCheck = %q, want actionable (intake incomplete)", truckB.Tasks["weightCheck"])
	}
	if truckB.Finished {
		t.Fatal("truck B intake must not be finished")
	}

	lotProc, ok := worklistJobByID(wl, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing worklist")
	}
	finished, ok := worklistInstance(lotProc, SeedLoadCOID)
	if !ok || !finished.Finished || finished.Tasks["lock"] != TaskStatusDone {
		t.Fatalf("finished lot %s = %#v, want lock done", SeedLoadCOID, finished)
	}
	openLot, ok := worklistInstance(lotProc, SeedLoadDAID)
	if !ok || openLot.Tasks["startLot"] != TaskStatusDone {
		t.Fatalf("open lot %s = %#v", SeedLoadDAID, openLot)
	}
	if openLot.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("open lot weigh = %q, want actionable after partial weigh", openLot.Tasks["weigh"])
	}
	if openLot.Finished {
		t.Fatal("open lot must not be finished")
	}
	if _, ok := worklistInstance(lotProc, SeedTruckBLotID); ok {
		t.Fatalf("truck B lot %s must not have lot_processing yet", SeedTruckBLotID)
	}

	if loads[SeedLoadCOID]["status"] != "fullyProcessed" || loads[SeedLoadCOID]["locked"] != true {
		t.Fatalf("finished lot facts = %#v, want fullyProcessed+locked", loads[SeedLoadCOID])
	}
	if loads[SeedLoadDAID]["status"] != "open" {
		t.Fatalf("open lot status = %#v, want open", loads[SeedLoadDAID])
	}
	if loads[SeedTruckBLotID]["status"] == "open" || loads[SeedTruckBLotID]["status"] == "fullyProcessed" {
		t.Fatalf("truck B lot status = %#v, want registered (not open/fullyProcessed)", loads[SeedTruckBLotID])
	}
	if _, ok := worklistJobByID(wl, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}

	list := ListStreamInstances(cfg, got, op)
	if len(list) != 2 {
		t.Fatalf("ListStreamInstances = %#v, want 2 trucks", list)
	}
	active := 0
	for _, row := range list {
		if row.Status == StreamInstanceActive {
			active++
		}
	}
	if active != 2 {
		t.Fatalf("active trucks = %d in %#v, want 2", active, list)
	}

	// /my truck A timeline: one finished lot (DPP) + one open mid-weigh row.
	tlFocus, err := TimelineInstance(SeedTruckID)
	if err != nil {
		t.Fatal(err)
	}
	tl := ProjectStreamTimeline(cfg, got, op, tlFocus)
	var each *StreamTimelineStep
	for i := range tl.Steps {
		if tl.Steps[i].JobID == "lot_processing" {
			each = &tl.Steps[i]
			break
		}
	}
	if each == nil || len(each.Rows) != 2 {
		t.Fatalf("truck A Lavorazione rows = %#v, want 2 (finished + open)", tl.Steps)
	}
	var finishedRow, openRow bool
	for _, row := range each.Rows {
		switch row.ID {
		case SeedLoadCOID:
			if !row.Finished {
				t.Fatalf("finished lot row %#v, want Finished", row)
			}
			finishedRow = true
		case SeedLoadDAID:
			if row.Finished {
				t.Fatalf("open lot row %#v must not be Finished", row)
			}
			openRow = true
		}
	}
	if !finishedRow || !openRow {
		t.Fatalf("truck A rows = %#v, want %s finished and %s open", each.Rows, SeedLoadCOID, SeedLoadDAID)
	}

	// Truck B Apri lotto stays open while DA is open on truck A (per-truck line-busy).
	tlBFocus, err := TimelineInstance(SeedTruckBID)
	if err != nil {
		t.Fatal(err)
	}
	tlB := ProjectStreamTimeline(cfg, got, op, tlBFocus)
	var startState StreamSubstepState
	var startBlocked string
	for _, step := range tlB.Steps {
		if step.JobID != "lot_processing" {
			continue
		}
		for _, row := range step.Rows {
			if row.ID != SeedTruckBLotID {
				continue
			}
			if len(row.Tasks) == 0 || row.Tasks[0].ID != "startLot" {
				t.Fatalf("truck B lot timeline tasks = %#v", row.Tasks)
			}
			startState = row.Tasks[0].State
			startBlocked = row.Tasks[0].BlockedReason
		}
	}
	if startState == StreamSubstepLocked {
		t.Fatalf("truck B Apri lotto state = %q, want not locked", startState)
	}
	if startBlocked != "" {
		t.Fatalf("truck B Apri lotto blockedReason = %q, want empty", startBlocked)
	}
}
