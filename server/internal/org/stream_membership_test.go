package org

import (
	"testing"
)

func TestJobInstanceBelongsToStreamAnchor(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "Truck A"},
		{JobID: "intake", ID: "T-B", Label: "Truck B"},
	}

	if !JobInstanceBelongsToStream(cfg, domain, "intake", "T-A", "T-A") {
		t.Fatal("anchor instance must belong to itself")
	}
	if JobInstanceBelongsToStream(cfg, domain, "intake", "T-B", "T-A") {
		t.Fatal("other anchor instance must not belong")
	}
}

func TestResolveStreamInstanceID(t *testing.T) {
	cfg := validStreamCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L-A1": {"truckId": "T-A"}},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "lot_processing", ID: "L-A1", Label: "L-A1"},
	}
	if got := ResolveStreamInstanceID(cfg, domain, "intake", "T-A"); got != "T-A" {
		t.Fatalf("anchor = %q, want T-A", got)
	}
	if got := ResolveStreamInstanceID(cfg, domain, "lot_processing", "L-A1"); got != "T-A" {
		t.Fatalf("each row = %q, want T-A", got)
	}
	if got := ResolveStreamInstanceID(cfg, domain, "lot_processing", ""); got != "" {
		t.Fatalf("provisional each start = %q, want empty", got)
	}
}

func TestEffectiveStreamInstanceID(t *testing.T) {
	cfg := validStreamCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L-A1": {"truckId": "T-A"}},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "lot_processing", ID: "L-A1", Label: "L-A1"},
	}
	if got := EffectiveStreamInstanceID(cfg, domain, "lot_processing", "L-A1", "ignored"); got != "T-A" {
		t.Fatalf("resolved wins = %q, want T-A", got)
	}
	if got := EffectiveStreamInstanceID(cfg, domain, "lot_processing", "", "T-A"); got != "T-A" {
		t.Fatalf("provisional fallback = %q, want T-A", got)
	}
	if got := EffectiveStreamInstanceID(cfg, domain, "lot_processing", "", "  T-A  "); got != "T-A" {
		t.Fatalf("trimmed fallback = %q, want T-A", got)
	}
	if got := EffectiveStreamInstanceID(cfg, domain, "lot_processing", "", ""); got != "" {
		t.Fatalf("unbound = %q, want empty", got)
	}
}

func TestResolveReadBindsStream(t *testing.T) {
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L-A1": {"truckId": "T-A"},
			"L-B1": {"truckId": "T-B"},
		},
	}
	ids, err := ResolveRead(domain, ReadDef{
		Entity: "lot",
		Where:  "truckId = $stream.id",
		Key:    "id",
	}, "", "", nil, &streamBinding{Key: "pv-recovery", ID: "T-A"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "L-A1" {
		t.Fatalf("ids = %#v, want [L-A1]", ids)
	}
}

func TestJobInstanceBelongsToStreamEachRows(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	putKind(&domain, "lot", EntityMap{
		"L-A1": {"truckId": "T-A", "status": "registered"},
		"L-A2": {"truckId": "T-A", "status": "registered"},
		"L-B1": {"truckId": "T-B", "status": "registered"},
	})
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "Truck A"},
		{JobID: "intake", ID: "T-B", Label: "Truck B"},
		{JobID: "lot_processing", ID: "L-A1", Label: "Lotto L-A1"},
		{JobID: "lot_processing", ID: "L-B1", Label: "Lotto L-B1"},
	}

	if !JobInstanceBelongsToStream(cfg, domain, "lot_processing", "L-A1", "T-A") {
		t.Fatal("each row for truck A lot must belong to T-A")
	}
	if !JobInstanceBelongsToStream(cfg, domain, "lot_processing", "L-A2", "T-A") {
		t.Fatal("each-selected entity id belongs even before a Job instance exists (start row)")
	}
	if JobInstanceBelongsToStream(cfg, domain, "lot_processing", "L-B1", "T-A") {
		t.Fatal("truck B lot must not belong to T-A")
	}
}
