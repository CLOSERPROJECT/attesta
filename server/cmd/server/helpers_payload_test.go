package main

import "testing"

func TestNormalizePayload(t *testing.T) {
	numberSub := WorkflowSub{InputType: "number", InputKey: "value"}
	payload, err := normalizePayload(numberSub, "10.5")
	if err != nil {
		t.Fatalf("expected number payload to parse, got error: %v", err)
	}
	if got, ok := payload["value"].(float64); !ok || got != 10.5 {
		t.Fatalf("expected parsed float 10.5, got %#v", payload["value"])
	}

	if _, err := normalizePayload(numberSub, "not-a-number"); err == nil {
		t.Fatal("expected number payload parse failure")
	}

	textSub := WorkflowSub{InputType: "text", InputKey: "note"}
	textPayload, err := normalizePayload(textSub, "ok")
	if err != nil {
		t.Fatalf("expected text payload to pass, got error: %v", err)
	}
	if got := textPayload["note"]; got != "ok" {
		t.Fatalf("expected text payload value "+`"ok"`+", got %#v", got)
	}
}

func TestDigestPayloadStability(t *testing.T) {
	left := map[string]interface{}{"b": "two", "a": 1.0}
	right := map[string]interface{}{"a": 1.0, "b": "two"}
	changed := map[string]interface{}{"a": 2.0, "b": "two"}

	leftDigest := digestPayload(left)
	rightDigest := digestPayload(right)
	changedDigest := digestPayload(changed)

	if leftDigest == "" {
		t.Fatal("expected non-empty digest")
	}
	if leftDigest != rightDigest {
		t.Fatalf("expected stable digest independent of map order: %q != %q", leftDigest, rightDigest)
	}
	if leftDigest == changedDigest {
		t.Fatalf("expected digest to change when payload changes: %q", leftDigest)
	}
}
