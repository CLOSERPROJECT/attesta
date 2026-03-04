package main

import (
	"net/url"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDPPSerialFromStrategyProcessIDHex(t *testing.T) {
	processID := primitive.NewObjectID()
	serial, err := dppSerialFromStrategy("process_id_hex", processID)
	if err != nil {
		t.Fatalf("dppSerialFromStrategy(process_id_hex): %v", err)
	}
	if serial == "" {
		t.Fatal("serial is empty")
	}
	if serial != processID.Hex() {
		t.Fatalf("serial = %q, want %q", serial, processID.Hex())
	}
	if escaped := url.PathEscape(serial); escaped != serial {
		t.Fatalf("serial is not URL-safe: escaped=%q serial=%q", escaped, serial)
	}
}

func TestNormalizeDPPSerialStrategyRejectsUnsupportedValue(t *testing.T) {
	_, err := normalizeDPPSerialStrategy("counter")
	if err == nil {
		t.Fatal("expected unsupported strategy error")
	}
}

func TestNormalizeDPPSerialStrategyDefaultsBlankInput(t *testing.T) {
	strategy, err := normalizeDPPSerialStrategy("   ")
	if err != nil {
		t.Fatalf("normalizeDPPSerialStrategy(blank): %v", err)
	}
	if strategy != "process_id_hex" {
		t.Fatalf("normalizeDPPSerialStrategy(blank) = %q, want %q", strategy, "process_id_hex")
	}
}

func TestGS1ElementStringFormatting(t *testing.T) {
	if got := gs1ElementString(" 09506000134352 ", " LOT-42 ", " SERIAL-9 "); got != "(01)09506000134352(10)LOT-42(21)SERIAL-9" {
		t.Fatalf("gs1ElementString() = %q", got)
	}
	if got := gs1ElementString("09506000134352", "", "SERIAL-9"); got != "" {
		t.Fatalf("gs1ElementString(missing lot) = %q, want empty", got)
	}
}
