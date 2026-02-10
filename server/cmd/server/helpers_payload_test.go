package main

import (
	"encoding/json"
	"testing"
)

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

type fixedStringer string

func (s fixedStringer) String() string {
	return string(s)
}

func TestAsString(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
		ok    bool
	}{
		{name: "string trim", value: "  hello  ", want: "hello", ok: true},
		{name: "stringer trim", value: fixedStringer("  hi  "), want: "hi", ok: true},
		{name: "number", value: 42, want: "42", ok: true},
		{name: "bool", value: true, want: "true", ok: true},
		{name: "nil", value: nil, want: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := asString(tc.value)
			if ok != tc.ok {
				t.Fatalf("asString(%#v) ok = %t, want %t", tc.value, ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("asString(%#v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestAsInt64(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int64
		ok    bool
	}{
		{name: "int", value: int(7), want: 7, ok: true},
		{name: "int32", value: int32(8), want: 8, ok: true},
		{name: "int64", value: int64(9), want: 9, ok: true},
		{name: "float32 truncated", value: float32(10.8), want: 10, ok: true},
		{name: "float64 truncated", value: float64(11.9), want: 11, ok: true},
		{name: "json number", value: json.Number("12"), want: 12, ok: true},
		{name: "json number invalid", value: json.Number("12.5"), want: 0, ok: false},
		{name: "string", value: " 13 ", want: 13, ok: true},
		{name: "string invalid", value: "abc", want: 0, ok: false},
		{name: "nil", value: nil, want: 0, ok: false},
		{name: "unsupported uint", value: uint(14), want: 0, ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := asInt64(tc.value)
			if ok != tc.ok {
				t.Fatalf("asInt64(%#v) ok = %t, want %t", tc.value, ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("asInt64(%#v) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}
