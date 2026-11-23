package org

import "testing"

func TestTimelineBoot(t *testing.T) {
	f := TimelineBoot()
	if !f.IsBoot() {
		t.Fatal("Boot should IsBoot")
	}
	if id, ok := f.InstanceID(); ok || id != "" {
		t.Fatalf("Boot InstanceID = (%q, %v)", id, ok)
	}
}

func TestTimelineInstance(t *testing.T) {
	f, err := TimelineInstance("T1")
	if err != nil {
		t.Fatalf("TimelineInstance: %v", err)
	}
	if f.IsBoot() {
		t.Fatal("Instance should not IsBoot")
	}
	id, ok := f.InstanceID()
	if !ok || id != "T1" {
		t.Fatalf("InstanceID = (%q, %v)", id, ok)
	}

	if _, err := TimelineInstance(""); err == nil {
		t.Fatal("blank TimelineInstance should err")
	}
	if _, err := TimelineInstance("  "); err == nil {
		t.Fatal("whitespace TimelineInstance should err")
	}
}

func TestResolveTimelineFocus(t *testing.T) {
	boot, err := ResolveTimelineFocus("")
	if err != nil {
		t.Fatalf("blank: %v", err)
	}
	if !boot.IsBoot() {
		t.Fatal("blank should resolve Boot")
	}

	bootWS, err := ResolveTimelineFocus("  ")
	if err != nil {
		t.Fatalf("whitespace: %v", err)
	}
	if !bootWS.IsBoot() {
		t.Fatal("whitespace should resolve Boot")
	}

	inst, err := ResolveTimelineFocus(" truck-a ")
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	id, ok := inst.InstanceID()
	if !ok || id != "truck-a" {
		t.Fatalf("resolved InstanceID = (%q, %v)", id, ok)
	}
	if inst.IsBoot() {
		t.Fatal("non-blank should not IsBoot")
	}
}
