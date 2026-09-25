package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Every stream config shipped in server/config is seeded verbatim into the
// store on first boot (bootstrapFormataBuilderStreams) and parsed back on every
// catalog load. A shipped config the loader rejects therefore breaks the whole
// catalog, not just its own stream, and it only fails at runtime.
func TestShippedStreamConfigsLoad(t *testing.T) {
	entries, err := os.ReadDir("../../config")
	if err != nil {
		t.Fatalf("read config dir: %v", err)
	}
	seen := 0
	for _, entry := range entries {
		if entry.IsDir() || !isWorkflowCatalogConfigFile(entry.Name()) {
			continue
		}
		seen++
		path := filepath.Join("../../config", entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		if _, parseErr := parseRuntimeConfigData(entry.Name(), data); parseErr != nil {
			t.Fatalf("%s does not load: %v", entry.Name(), parseErr)
		}
	}
	if seen == 0 {
		t.Fatal("no shipped stream configs found")
	}
}
