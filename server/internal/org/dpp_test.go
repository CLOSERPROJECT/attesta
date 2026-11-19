package org

import (
	"errors"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

func syntheticDPPConfig() OrgConfig {
	lock := TaskConfig{
		"id":    "lock",
		"roles": []any{"operator"},
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ack": map[string]any{"type": "boolean"},
			},
		},
		"onSubmit": map[string]any{
			"transform": []any{
				map[string]any{
					"op":     "patch",
					"entity": "widget",
					"id":     "$instance.id",
					"set": map[string]any{
						"locked": true,
					},
				},
			},
			"passport": map[string]any{},
		},
	}
	return OrgConfig{
		ID:      "passport-synth",
		Org:     "synth",
		Version: "v1",
		Stream:  streamForAnchor("synth", "Synth", "issue"),
		DPP: &DPPConfig{
			Subject: DPPSubjectConfig{Entity: "widget"},
			Identity: DPPIdentityConfig{
				GTIN:        `"9506000134352"`,
				Lot:         `$lookup($domain.widget, $subjectId).lot`,
				Serial:      `$subjectId`,
				ProductName: `"Synth Widget"`,
			},
			Snapshot: DPPSnapshotConfig{
				// dataReport-shaped so Lock Load chrome paints the same object Commit freezes.
				Expr: `{
					"title": "Passport",
					"rows": [
						{"label": "Subject", "value": $subjectId, "error": null, "tone": null},
						{"label": "Color", "value": $string($lookup($domain.widget, $subjectId).color), "error": null, "tone": null},
						{"label": "Lot", "value": $lookup($domain.widget, $subjectId).lot, "error": null, "tone": null}
					],
					"subjectId": $subjectId,
					"color": $lookup($domain.widget, $subjectId).color,
					"lot": $lookup($domain.widget, $subjectId).lot
				}`,
			},
		},
		Jobs: OrderedObject{
			Keys: []string{"issue"},
			Values: map[string]any{
				"issue": Job{
					Repeatable: true,
					InstanceID: "$form.widgetId",
					Label:      `"Widget " & $form.widgetId`,
					Tasks:      []TaskConfig{lock},
				},
			},
		},
	}
}

func TestValidatePassportConfigRequiresRootWhenOptIn(t *testing.T) {
	cfg := syntheticDPPConfig()
	cfg.DPP = nil
	if err := ValidateDPPConfig(cfg); err == nil {
		t.Fatal("expected opt-in without root passport to fail")
	}
}

func TestValidatePassportConfigRejectsIncompleteBlock(t *testing.T) {
	cfg := syntheticDPPConfig()
	cfg.DPP.Snapshot.Expr = ""
	if err := ValidateDPPConfig(cfg); err == nil {
		t.Fatal("expected empty snapshot expr to fail")
	}
}

func TestMemoryPassportStoreCreateOnce(t *testing.T) {
	store := NewMemoryDPPStore()
	rec := DPPRecord{
		Org: "synth", SubjectID: "W1", GTIN: "09506000134352",
		Lot: "L1", Serial: "W1", Snapshot: map[string]any{"n": 1}, IssuedAt: "t",
		DigitalLink: untp.URL("09506000134352", "L1", "W1"),
	}
	if err := store.Create(t.Context(), rec); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "synth", "W1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Lot != "L1" || got.Snapshot["n"] != 1 {
		t.Fatalf("got = %#v", got)
	}
	if err := store.Create(t.Context(), rec); !errors.Is(err, ErrDPPExists) {
		t.Fatalf("second create = %v", err)
	}
}

func TestCommitPassportProducerIssuesOnce(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryDPPStore()
	cfg := syntheticDPPConfig()
	if err := configs.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1"); err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{
		"W-9": {"lot": "LOT-9", "color": "blue"},
	})
	domain.JobInstances = []JobInstance{{JobID: "issue", ID: "W-9", Label: "Widget W-9"}}
	if err := domains.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, domain); err != nil {
		t.Fatal(err)
	}

	c := &TaskCommitter{
		Domains:    domains,
		Configs:    configs,
		DPPs:       passports,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) },
	}
	op := OperatorIdentity{UserID: "ada", Org: "synth", Roles: []string{"operator"}}
	result, err := c.Commit(t.Context(), Scope{Org: "synth", Stream: "synth"}, op, CommitRequest{
		TaskPath:      "issue/lock",
		JobInstanceID: "W-9",
		Value:         map[string]any{"ack": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("widget")["W-9"]["locked"] != true {
		t.Fatalf("transform locked fact missing: %#v", result.Domain.Kind("widget")["W-9"])
	}
	if len(result.Links) != 1 {
		t.Fatalf("links = %#v", result.Links)
	}
	wantLink := untp.URL("09506000134352", "LOT-9", "W-9")
	if result.Links[0].URL != wantLink || result.Links[0].Action != HttpActionLink {
		t.Fatalf("link = %#v, want %q", result.Links[0], wantLink)
	}

	rec, err := passports.Get(t.Context(), "synth", "W-9")
	if err != nil {
		t.Fatal(err)
	}
	if rec.GTIN != "09506000134352" || rec.Lot != "LOT-9" || rec.Serial != "W-9" {
		t.Fatalf("identity = %#v", rec)
	}
	if rec.ProductName != "Synth Widget" {
		t.Fatalf("productName = %q", rec.ProductName)
	}
	if rec.Snapshot["subjectId"] != "W-9" || rec.Snapshot["color"] != "blue" || rec.Snapshot["lot"] != "LOT-9" {
		t.Fatalf("snapshot = %#v", rec.Snapshot)
	}
	if rec.Snapshot["title"] != "Passport" {
		t.Fatalf("snapshot title = %#v", rec.Snapshot["title"])
	}
	if rec.DigitalLink != wantLink {
		t.Fatalf("digitalLink = %q", rec.DigitalLink)
	}

	_, err = c.Commit(t.Context(), Scope{Org: "synth", Stream: "synth"}, op, CommitRequest{
		TaskPath:      "issue/lock",
		JobInstanceID: "W-9",
		Value:         map[string]any{"ack": true},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "passport-exists" {
		t.Fatalf("second lock err = %v", err)
	}
}

func TestCommitPassportRequiresRootBlock(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := syntheticDPPConfig()
	cfg.DPP = nil
	if err := configs.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1"); err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{"W-1": {"lot": "L"}})
	domain.JobInstances = []JobInstance{{JobID: "issue", ID: "W-1", Label: "W-1"}}
	_ = domains.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, domain)

	c := &TaskCommitter{
		Domains: domains, Configs: configs, DPPs: NewMemoryDPPStore(),
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) },
	}
	_, err := c.Commit(t.Context(), Scope{Org: "synth", Stream: "synth"}, OperatorIdentity{UserID: "ada", Org: "synth", Roles: []string{"operator"}}, CommitRequest{
		TaskPath: "issue/lock", JobInstanceID: "W-1", Value: map[string]any{"ack": true},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
}

func TestCommitWithoutPassportOptInSkipsStore(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryDPPStore()
	cfg := syntheticDPPConfig()
	lock := cfg.Jobs.Values["issue"].(Job).Tasks[0]
	onSubmit := lock["onSubmit"].(map[string]any)
	delete(onSubmit, "passport")
	if err := configs.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), Scope{Org: "synth", Stream: "synth"}, "v1"); err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{"W-2": {"lot": "L2"}})
	domain.JobInstances = []JobInstance{{JobID: "issue", ID: "W-2", Label: "W-2"}}
	_ = domains.Put(t.Context(), Scope{Org: "synth", Stream: "synth"}, domain)

	c := &TaskCommitter{
		Domains: domains, Configs: configs, DPPs: passports,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) },
	}
	result, err := c.Commit(t.Context(), Scope{Org: "synth", Stream: "synth"}, OperatorIdentity{UserID: "ada", Org: "synth", Roles: []string{"operator"}}, CommitRequest{
		TaskPath: "issue/lock", JobInstanceID: "W-2", Value: map[string]any{"ack": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Links) != 0 {
		t.Fatalf("links = %#v", result.Links)
	}
	if _, err := passports.Get(t.Context(), "synth", "W-2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("store get = %v", err)
	}
}
