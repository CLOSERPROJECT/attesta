package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"
)

func minimalWorkflowYAML(name string) string {
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
	return "workflow:\n" +
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
}

func saveFormataStream(t *testing.T, store Store, name string) string {
	t.Helper()
	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalWorkflowYAML(name),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream %s: %v", name, err)
	}
	return saved.ID.Hex()
}

func TestSeedInstancesWithStoreOpenerSeedsCatalog(t *testing.T) {
	dir := t.TempDir()
	store := NewMemoryStore()
	key := saveFormataStream(t, store, "Demo Stream")
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir)
	if err != nil {
		t.Fatalf("seedInstancesWithStoreOpener: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), key, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 8 {
		t.Fatalf("demo instances = %d, want 8", len(list))
	}
}

func TestSeedInstancesWithStoreOpenerLeavesEveryFifthStreamEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewMemoryStore()
	keys := make([]string, 0, 5)
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		keys = append(keys, saveFormataStream(t, store, "Stream "+name))
	}
	sort.Strings(keys)
	emptyKey := keys[4]
	store.SeedProcess(Process{WorkflowKey: emptyKey, Status: processStatusActive, CreatedAt: time.Now().UTC()})
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir)
	if err != nil {
		t.Fatalf("seedInstancesWithStoreOpener: %v", err)
	}
	for _, key := range keys[:4] {
		list, listErr := store.ListRecentProcessesByWorkflow(t.Context(), key, 0)
		if listErr != nil {
			t.Fatalf("list %s: %v", key, listErr)
		}
		if len(list) != 8 {
			t.Fatalf("%s instances = %d, want 8", key, len(list))
		}
	}
	emptyList, err := store.ListRecentProcessesByWorkflow(t.Context(), emptyKey, 0)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(emptyList) != 0 {
		t.Fatalf("empty-key instances = %d, want 0 (empty-state stream)", len(emptyList))
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
