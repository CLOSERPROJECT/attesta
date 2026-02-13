package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParseDigitalLinkPathValidAndInvalid(t *testing.T) {
	gtin, lot, serial, err := parseDigitalLinkPath("/01/09506000134352/10/LOT-001/21/SERIAL-001")
	if err != nil {
		t.Fatalf("parseDigitalLinkPath(valid): %v", err)
	}
	if gtin != "09506000134352" || lot != "LOT-001" || serial != "SERIAL-001" {
		t.Fatalf("unexpected parsed values: gtin=%q lot=%q serial=%q", gtin, lot, serial)
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/LOT-001")
	if err == nil {
		t.Fatal("expected invalid path shape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/not-digits/10/LOT-001/21/SERIAL-001")
	if err == nil {
		t.Fatal("expected invalid gtin error")
	}
}

func TestDigitalLinkURLPathEscapesValues(t *testing.T) {
	url := digitalLinkURL("09506000134352", "LOT 001", "SERIAL/001")
	if url != "/01/09506000134352/10/LOT%20001/21/SERIAL%2F001" {
		t.Fatalf("digitalLinkURL() = %q", url)
	}
}

func TestDPPFirstStringValueAndBuildProcessDPP(t *testing.T) {
	def := testRuntimeConfig().Workflow
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
			"1.2": {State: "done", Data: map[string]interface{}{"note": "LOT-2026"}},
			"3.2": {State: "done", Data: map[string]interface{}{"serialCode": "SER-ABC"}},
		},
	}

	if got := dppFirstStringValue(def, process, "note"); got != "LOT-2026" {
		t.Fatalf("dppFirstStringValue(note) = %q, want LOT-2026", got)
	}
	if got := dppFirstStringValue(def, process, "missing"); got != "" {
		t.Fatalf("dppFirstStringValue(missing) = %q, want empty", got)
	}

	cfg := DPPConfig{
		Enabled:        true,
		GTIN:           "09506000134352",
		LotInputKey:    "note",
		SerialInputKey: "serialCode",
		SerialStrategy: "process_id_hex",
		LotDefault:     "defaultProduct",
	}
	now := time.Date(2026, 2, 13, 11, 0, 0, 0, time.UTC)
	dpp, err := buildProcessDPP(def, cfg, process, now)
	if err != nil {
		t.Fatalf("buildProcessDPP: %v", err)
	}
	if dpp.GTIN != cfg.GTIN || dpp.Lot != "LOT-2026" || dpp.Serial != "SER-ABC" {
		t.Fatalf("unexpected dpp: %#v", dpp)
	}

	cfg.SerialInputKey = "missing"
	dpp, err = buildProcessDPP(def, cfg, process, now)
	if err != nil {
		t.Fatalf("buildProcessDPP fallback serial: %v", err)
	}
	if dpp.Serial != process.ID.Hex() {
		t.Fatalf("serial fallback = %q, want %q", dpp.Serial, process.ID.Hex())
	}
}

func TestBuildProcessDPPErrorsAndStrategyValidation(t *testing.T) {
	def := testRuntimeConfig().Workflow
	now := time.Date(2026, 2, 13, 11, 0, 0, 0, time.UTC)
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
		},
	}

	cfg := DPPConfig{
		Enabled:        true,
		GTIN:           "09506000134352",
		LotInputKey:    "note",
		SerialInputKey: "serialCode",
		SerialStrategy: "process_id_hex",
	}

	if _, err := buildProcessDPP(def, cfg, nil, now); err == nil {
		t.Fatal("expected error for nil process")
	}

	cfg.Enabled = false
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected error when dpp is disabled")
	}
	cfg.Enabled = true

	cfg.GTIN = ""
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected missing gtin error")
	}
	cfg.GTIN = "09506000134352"

	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected missing lot error")
	}

	cfg.LotDefault = "LOT-DEFAULT"
	cfg.SerialStrategy = "unsupported"
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected unsupported serial strategy error")
	}
}

func TestParseDigitalLinkPathUnescapeErrors(t *testing.T) {
	_, _, _, err := parseDigitalLinkPath("/01/09506000134352/10/%ZZ/21/SERIAL-001")
	if err == nil {
		t.Fatal("expected lot unescape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/LOT-001/21/%ZZ")
	if err == nil {
		t.Fatal("expected serial unescape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/ /21/SERIAL-001")
	if err == nil {
		t.Fatal("expected missing lot or serial error")
	}
}

func TestBuildDPPTraceabilityViewIncludesValuesAndFiles(t *testing.T) {
	def := testRuntimeConfig().Workflow
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data:  map[string]interface{}{"value": float64(10)},
			},
			"1.2": {
				State: "done",
				Data:  map[string]interface{}{"note": "LOT-2026"},
			},
			"1.3": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": "65f2a79b8e7f7d8f3c7c99aa",
						"filename":     "cert.pdf",
						"sha256":       "abc123",
					},
				},
			},
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow")
	if len(view) == 0 {
		t.Fatal("expected non-empty traceability view")
	}

	var foundValue bool
	var foundFile bool
	for _, step := range view {
		for _, sub := range step.Substeps {
			if sub.SubstepID == "1.2" {
				for _, value := range sub.Values {
					if value.Key == "note" && value.Value == "LOT-2026" {
						foundValue = true
					}
				}
			}
			if sub.SubstepID == "1.3" {
				if sub.FileName == "cert.pdf" && sub.FileURL != "" {
					foundFile = true
				}
			}
		}
	}
	if !foundValue {
		t.Fatal("expected traceability value entry for substep 1.2")
	}
	if !foundFile {
		t.Fatal("expected inline file metadata for substep 1.3")
	}
}
