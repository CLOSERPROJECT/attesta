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
