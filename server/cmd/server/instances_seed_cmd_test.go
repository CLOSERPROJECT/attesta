package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeMinimalWorkflowYAML(t *testing.T, path, name string) {
	t.Helper()
	substepLines := ""
	for i := 1; i <= 5; i++ {
		id := "1." + string(rune('0'+i))
		key := string(rune('a' + i - 1))
		substepLines += "        - id: \"" + id + "\"\n" +
			"          title: \"Input " + string(rune('A'+i-1)) + "\"\n" +
			"          order: " + fmt.Sprintf("%d", i) + "\n" +
			"          roles: [\"dep1\"]\n" +
			"          inputKey: \"" + key + "\"\n" +
			"          inputType: \"formata\"\n" +
			"          schema:\n" +
			"            type: object\n"
	}
	content := "workflow:\n" +
		"  name: \"" + name + "\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      organization: \"org1\"\n" +
		"      substeps:\n" +
		substepLines +
		"organizations:\n" +
		"  - slug: \"org1\"\n" +
		"    name: \"Organization 1\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org1\"\n" +
		"    slug: \"dep1\"\n" +
		"    name: \"Department 1\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config %s: %v", path, err)
	}
}

func seedInstancesViaSession(ctx context.Context, store Store, configDir string) error {
	return withSeedSession(ctx, nil, configDir, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, nil, func(sess seedSession) error {
		catalog, err := loadWorkflowCatalog(ctx, sess.Store, sess.ConfigDir)
		if err != nil {
			return fmt.Errorf("instance seed: catalog: %w", err)
		}
		_, err = seedDemoInstances(ctx, sess.Store, catalog, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("instance seed: %w", err)
		}
		return nil
	})
}

func TestWithSeedSessionSeedsInstances(t *testing.T) {
	dir := t.TempDir()
	writeMinimalWorkflowYAML(t, filepath.Join(dir, "demo.yaml"), "Demo Stream")
	store := NewMemoryStore()
	if err := seedInstancesViaSession(t.Context(), store, dir); err != nil {
		t.Fatalf("withSeedSession: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "demo", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 8 {
		t.Fatalf("demo instances = %d, want 8", len(list))
	}
}

func TestWithSeedSessionLeavesEveryFifthStreamEmpty(t *testing.T) {
	dir := t.TempDir()
	// Sorted keys: a, b, c, d, e — e is 5th (index 4) and stays empty.
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		writeMinimalWorkflowYAML(t, filepath.Join(dir, name+".yaml"), "Stream "+name)
	}
	store := NewMemoryStore()
	// Pre-seed junk on the empty target so replace wipe is visible.
	store.SeedProcess(Process{WorkflowKey: "e", Status: processStatusActive, CreatedAt: time.Now().UTC()})
	if err := seedInstancesViaSession(t.Context(), store, dir); err != nil {
		t.Fatalf("withSeedSession: %v", err)
	}
	for _, key := range []string{"a", "b", "c", "d"} {
		list, listErr := store.ListRecentProcessesByWorkflow(t.Context(), key, 0)
		if listErr != nil {
			t.Fatalf("list %s: %v", key, listErr)
		}
		if len(list) != 8 {
			t.Fatalf("%s instances = %d, want 8", key, len(list))
		}
	}
	emptyList, err := store.ListRecentProcessesByWorkflow(t.Context(), "e", 0)
	if err != nil {
		t.Fatalf("list e: %v", err)
	}
	if len(emptyList) != 0 {
		t.Fatalf("e instances = %d, want 0 (empty-state stream)", len(emptyList))
	}
}

func TestWithSeedSessionInstancesEmptyCatalog(t *testing.T) {
	dir := t.TempDir()
	store := NewMemoryStore()
	if err := seedInstancesViaSession(t.Context(), store, dir); err == nil {
		t.Fatal("expected empty catalog error")
	}
}

func TestWithSeedSessionInstancesErrors(t *testing.T) {
	if err := withSeedSession(t.Context(), nil, t.TempDir(), nil, nil, func(seedSession) error { return nil }); err == nil {
		t.Fatal("expected nil opener error")
	}
	openErr := errors.New("open failed")
	if err := withSeedSession(t.Context(), nil, t.TempDir(), func(context.Context) (Store, func(), error) {
		return nil, nil, openErr
	}, nil, func(seedSession) error { return nil }); !errors.Is(err, openErr) {
		t.Fatalf("err = %v, want openErr", err)
	}
}

func TestRunSeedInstancesCommandUsesOpener(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	err := runSeedInstancesCommand(t.Context(), nil)
	if err == nil {
		t.Fatal("expected mongo connect/ping failure")
	}
}
