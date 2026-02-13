package main

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// dppSerialFromStrategy returns a deterministic serial using the configured strategy.
func dppSerialFromStrategy(strategy string, processID primitive.ObjectID) (string, error) {
	normalized, err := normalizeDPPSerialStrategy(strategy)
	if err != nil {
		return "", err
	}
	switch normalized {
	case "process_id_hex":
		return processID.Hex(), nil
	default:
		return "", fmt.Errorf("unsupported dpp.serialStrategy %q (allowed: process_id_hex)", strategy)
	}
}
