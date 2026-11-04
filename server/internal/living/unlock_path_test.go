package living

import "testing"

func intakePathCfg() OrgConfig {
	return OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{Repeatable: true, Tasks: intakeWithLotsTasks()},
			},
		},
	}
}

func TestEvaluatePathUnlockFlatLeaf(t *testing.T) {
	cfg := intakePathCfg()
	empty := EmptyDomain()

	first := EvaluatePathUnlock(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "")
	if !first.Unlocked || first.Done {
		t.Fatalf("first leaf = %#v", first)
	}

	lotsLocked := EvaluatePathUnlock(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "lots"}, "T1")
	if lotsLocked.Unlocked || lotsLocked.Done {
		t.Fatalf("lots before arrive = %#v", lotsLocked)
	}

	afterArrive := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	lotsOpen := EvaluatePathUnlock(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "lots"}, "T1")
	if !lotsOpen.Unlocked || lotsOpen.Done {
		t.Fatalf("lots after arrive = %#v", lotsOpen)
	}

	arriveDone := EvaluatePathUnlock(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "T1")
	if !arriveDone.Unlocked || !arriveDone.Done {
		t.Fatalf("truckArrive submitted = %#v", arriveDone)
	}
}

func TestEvaluatePathUnlockGroupClosedAndFollower(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{
		Submissions: []Submission{sub("intake", "truckArrive", "T1")},
	}

	weight := EvaluatePathUnlock(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1")
	if weight.Unlocked {
		t.Fatalf("weightCheck before close = %#v", weight)
	}

	domain.ClosedGroups = []ClosedGroup{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots"}}
	lotsDone := EvaluatePathUnlock(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "lots"}, "T1")
	if !lotsDone.Unlocked || !lotsDone.Done {
		t.Fatalf("lots closed = %#v", lotsDone)
	}
	weight = EvaluatePathUnlock(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1")
	if !weight.Unlocked || weight.Done {
		t.Fatalf("weightCheck after close = %#v", weight)
	}
}

func TestEvaluatePathUnlockProvisionalFirstChild(t *testing.T) {
	if ProvisionalGroupInstanceID != "new" {
		t.Fatalf("ProvisionalGroupInstanceID = %q, want new", ProvisionalGroupInstanceID)
	}
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	first := EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: ProvisionalGroupInstanceID, TaskID: "register",
	}, "T1")
	if !first.Unlocked || first.Done {
		t.Fatalf("provisional first child = %#v", first)
	}

	viaPath, err := ParseTaskPath("intake/lots/new/register")
	if err != nil {
		t.Fatal(err)
	}
	if viaPath.GroupInstanceID != ProvisionalGroupInstanceID {
		t.Fatalf("parsed provisional = %#v", viaPath)
	}
	parsed := EvaluatePathUnlock(cfg, domain, viaPath, "T1")
	if !parsed.Unlocked || parsed.Done {
		t.Fatalf("parsed new path = %#v", parsed)
	}

	legacy := EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "_new", TaskID: "register",
	}, "T1")
	if legacy.Unlocked {
		t.Fatalf("_new must not unlock as provisional = %#v", legacy)
	}

	cfgTwo := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{Tasks: []TaskConfig{
					{"id": "truckArrive"},
					{
						"id": "lots", "kind": "group",
						"tasks": []TaskConfig{{"id": "prepare"}, {"id": "register"}},
					},
				}},
			},
		},
	}
	second := EvaluatePathUnlock(cfgTwo, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: ProvisionalGroupInstanceID, TaskID: "register",
	}, "T1")
	if second.Unlocked {
		t.Fatalf("provisional non-first child = %#v", second)
	}
}

func TestEvaluatePathUnlockNestedInstanceAndGroupClosed(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{Tasks: []TaskConfig{
					{"id": "truckArrive"},
					{
						"id": "lots", "kind": "group",
						"tasks": []TaskConfig{{"id": "prepare"}, {"id": "register"}},
					},
					{"id": "weightCheck"},
				}},
			},
		},
	}
	domain := Domain{
		Submissions:    []Submission{sub("intake", "truckArrive", "T1")},
		GroupInstances: []GroupInstance{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots", ID: "L1"}},
	}

	prep := EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "L1", TaskID: "prepare",
	}, "T1")
	if !prep.Unlocked || prep.Done {
		t.Fatalf("first child open = %#v", prep)
	}
	reg := EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "L1", TaskID: "register",
	}, "T1")
	if reg.Unlocked {
		t.Fatalf("second child locked = %#v", reg)
	}

	domain.Submissions = append(domain.Submissions, Submission{
		JobID: "intake", TaskID: "prepare", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L1",
		Timestamp: "2026-11-02T10:00:00Z", UserID: "ada", Facts: map[string]any{},
	})
	reg = EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "L1", TaskID: "register",
	}, "T1")
	if !reg.Unlocked || reg.Done {
		t.Fatalf("second child after prepare = %#v", reg)
	}

	domain.Submissions = append(domain.Submissions, Submission{
		JobID: "intake", TaskID: "register", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L1",
		Timestamp: "2026-11-02T10:01:00Z", UserID: "ada", Facts: map[string]any{},
	})
	reg = EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "L1", TaskID: "register",
	}, "T1")
	if !reg.Unlocked || !reg.Done {
		t.Fatalf("register submitted = %#v", reg)
	}

	domain.ClosedGroups = []ClosedGroup{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots"}}
	reg = EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: "L1", TaskID: "register",
	}, "T1")
	if reg.Unlocked || !reg.Done {
		t.Fatalf("register after group close = %#v", reg)
	}
}

func TestSequenceOkForPathUsesUnlockedHalf(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	if sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk before close")
	}
	domain.ClosedGroups = []ClosedGroup{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots"}}
	if !sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk after lots closed")
	}
	if sequenceOkForPath(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: ProvisionalGroupInstanceID, TaskID: "register",
	}, "T1") {
		t.Fatal("closed group must deny provisional sequenceOk")
	}
}

func TestSequenceOkForPathProvisionalWhenGroupOpen(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	if !sequenceOkForPath(cfg, domain, TaskPathRef{
		JobID: "intake", GroupID: "lots", GroupInstanceID: ProvisionalGroupInstanceID, TaskID: "register",
	}, "T1") {
		t.Fatal("provisional first child sequenceOk while lots open")
	}
}

func TestProjectStatusAdaptersMapPathUnlock(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}

	if got := projectTaskStatus(cfg, domain, "intake", "truckArrive", "T1"); got != TaskStatusDone {
		t.Fatalf("truckArrive status = %q", got)
	}
	if got := projectGroupStatus(cfg, domain, "intake", "lots", "T1"); got != TaskStatusAvailable {
		t.Fatalf("lots status = %q", got)
	}
	domain.GroupInstances = []GroupInstance{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots", ID: "L1"}}
	if got := projectGroupChildStatus(cfg, domain, "intake", "T1", "lots", "L1", "register"); got != TaskStatusAvailable {
		t.Fatalf("register status = %q", got)
	}
}
