package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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

func TestSeedInstancesWithStoreOpenerSeedsCatalog(t *testing.T) {
	dir := t.TempDir()
	writeMinimalWorkflowYAML(t, filepath.Join(dir, "demo.yaml"), "Demo Stream")
	store := NewMemoryStore()
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir)
	if err != nil {
		t.Fatalf("seedInstancesWithStoreOpener: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "demo", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 8 {
		t.Fatalf("demo instances = %d, want 8", len(list))
	}
}

func TestSeedInstancesWithStoreOpenerEmptyCatalog(t *testing.T) {
	dir := t.TempDir()
	store := NewMemoryStore()
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir)
	if err == nil {
		t.Fatal("expected empty catalog error")
	}
}

func TestSeedInstancesWithStoreOpenerErrors(t *testing.T) {
	if err := seedInstancesWithStoreOpener(t.Context(), nil, nil, t.TempDir()); err == nil {
		t.Fatal("expected nil opener error")
	}
	openErr := errors.New("open failed")
	if err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return nil, nil, openErr
	}, t.TempDir()); !errors.Is(err, openErr) {
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
