package org

import (
	"errors"
	"testing"
)

func TestStreamsKeepSeparateDomainsAndActiveVersions(t *testing.T) {
	configs := NewMemoryConfigStore()
	domains := NewMemoryDomainStore()

	recycling, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other, err := LoadSecondStreamConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other.Org = recycling.Org

	recScope, err := ScopeFromConfig(recycling)
	if err != nil {
		t.Fatal(err)
	}
	otherScope, err := ScopeFromConfig(other)
	if err != nil {
		t.Fatal(err)
	}
	if recScope.Org != otherScope.Org || recScope.Stream == otherScope.Stream {
		t.Fatalf("scopes = %#v %#v", recScope, otherScope)
	}

	if err := configs.Put(t.Context(), recScope, "1", recycling); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), recScope, "1"); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), recScope, domainFromKinds(map[string]EntityMap{
		"lot": {"R1": {"client": "CO"}},
	})); err != nil {
		t.Fatal(err)
	}

	if err := configs.Put(t.Context(), otherScope, "9", other); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), otherScope, "9"); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), otherScope, domainFromKinds(map[string]EntityMap{
		"item": {"O1": {"label": "other"}},
	})); err != nil {
		t.Fatal(err)
	}

	activeRec, err := configs.ActiveVersion(t.Context(), recScope)
	if err != nil || activeRec != "1" {
		t.Fatalf("recycling active = %q err=%v", activeRec, err)
	}
	activeOther, err := configs.ActiveVersion(t.Context(), otherScope)
	if err != nil || activeOther != "9" {
		t.Fatalf("other active = %q err=%v", activeOther, err)
	}

	recDomain, err := domains.Get(t.Context(), recScope)
	if err != nil {
		t.Fatal(err)
	}
	if recDomain.Kind("lot")["R1"]["client"] != "CO" || len(recDomain.Kind("item")) != 0 {
		t.Fatalf("recycling domain leaked: %#v", recDomain)
	}
	otherDomain, err := domains.Get(t.Context(), otherScope)
	if err != nil {
		t.Fatal(err)
	}
	if otherDomain.Kind("item")["O1"]["label"] != "other" || len(otherDomain.Kind("lot")) != 0 {
		t.Fatalf("other domain leaked: %#v", otherDomain)
	}

	streams, err := configs.ListStreams(t.Context(), recycling.Org)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 2 || streams[0] != "other" || streams[1] != "recycling" {
		t.Fatalf("streams = %#v", streams)
	}
}

func TestRequireSoleStream(t *testing.T) {
	configs := NewMemoryConfigStore()
	if _, err := RequireSoleStream(t.Context(), configs, "feragame"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty = %v", err)
	}
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	scope, err := RequireSoleStream(t.Context(), configs, "feragame")
	if err != nil || scope != FeragameScope() {
		t.Fatalf("sole = %#v err=%v", scope, err)
	}

	other, err := LoadSecondStreamConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other.Org = "feragame"
	otherScope, err := ScopeFromConfig(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), otherScope, other.Version, other); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), otherScope, other.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := RequireSoleStream(t.Context(), configs, "feragame"); !errors.Is(err, ErrAmbiguousStreams) {
		t.Fatalf("ambiguous = %v", err)
	}
}
