package org

import (
	"strings"
	"testing"
)

func validStreamCfg() OrgConfig {
	return OrgConfig{
		ID:      "stream-synth",
		Org:     "synth",
		Version: "v1",
		Domain: map[string]any{
			"entities": map[string]any{
				"lot": map[string]any{"fields": []any{"truckId"}},
			},
		},
		Stream: &StreamConfig{
			Key:   "pv-recovery",
			Title: "Recupero materiali da pannelli fotovoltaici",
			Instance: StreamInstance{
				Job: "intake",
			},
			Steps: []StreamStep{
				{Job: "intake"},
				{
					Job: "lot_processing",
					Each: &StreamEach{
						Entity: "lot",
						Where:  "truckId = $stream.id",
					},
				},
			},
		},
		Jobs: OrderedObject{
			Keys: []string{"intake", "lot_processing"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					InstanceID: "$truck.id",
					Label:      "$truck.id",
					Tasks:      []TaskConfig{{"id": "arrive"}},
				},
				"lot_processing": Job{
					Repeatable: true,
					InstanceID: "$.lotId",
					Label:      "$.lotId",
					Tasks:      []TaskConfig{{"id": "start"}},
				},
			},
		},
	}
}

func TestValidateStreamConfigAcceptsValid(t *testing.T) {
	if err := ValidateStreamConfig(validStreamCfg()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStreamConfigRequiresStream(t *testing.T) {
	cfg := validStreamCfg()
	cfg.Stream = nil
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "stream") {
		t.Fatalf("err=%v, want stream required", err)
	}
}

func TestValidateStreamConfigRejectsReservedKeys(t *testing.T) {
	for _, key := range []string{"i", "new"} {
		cfg := validStreamCfg()
		cfg.Stream.Key = key
		err := ValidateStreamConfig(cfg)
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Fatalf("key=%q err=%v, want reserved rejection", key, err)
		}
	}
}

func TestValidateStreamConfigRejectsNonSlugKey(t *testing.T) {
	cfg := validStreamCfg()
	cfg.Stream.Key = "Bad Key"
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("err=%v, want slug rejection", err)
	}
}

func TestValidateStreamConfigRequiresRepeatableAnchorInSteps(t *testing.T) {
	cfg := validStreamCfg()
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Repeatable = false
	cfg.Jobs.Values["intake"] = intake
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "repeatable") {
		t.Fatalf("err=%v, want repeatable anchor", err)
	}

	cfg = validStreamCfg()
	cfg.Stream.Steps = []StreamStep{{Job: "lot_processing", Each: &StreamEach{Entity: "lot", Where: "true"}}}
	err = ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "steps") {
		t.Fatalf("err=%v, want anchor listed in steps", err)
	}
}

func TestValidateStreamConfigRejectsUnknownStepJob(t *testing.T) {
	cfg := validStreamCfg()
	cfg.Stream.Steps = append(cfg.Stream.Steps, StreamStep{Job: "missing"})
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("err=%v, want unknown step job", err)
	}
}

func TestValidateStreamConfigRejectsUndeclaredEachEntity(t *testing.T) {
	cfg := validStreamCfg()
	cfg.Stream.Steps[1].Each.Entity = "pallet"
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "pallet") {
		t.Fatalf("err=%v, want undeclared entity", err)
	}
}

func TestValidateStreamConfigRejectsBadEachWhere(t *testing.T) {
	cfg := validStreamCfg()
	cfg.Stream.Steps[1].Each.Where = "(("
	err := ValidateStreamConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "where") {
		t.Fatalf("err=%v, want where compile failure", err)
	}
}

func TestValidateTaskConfigsRejectsBadBlockedReason(t *testing.T) {
	err := validateTaskConfigs([]TaskConfig{
		{"id": "open", "blockedReason": "1 + ("},
	})
	if err == nil || !strings.Contains(err.Error(), "blockedReason") {
		t.Fatalf("bad blockedReason = %v", err)
	}
}

func TestValidateTaskConfigsAcceptsBlockedReason(t *testing.T) {
	if err := validateTaskConfigs([]TaskConfig{
		{"id": "open", "blockedReason": `'busy: ' & $id`},
	}); err != nil {
		t.Fatalf("valid blockedReason: %v", err)
	}
}
