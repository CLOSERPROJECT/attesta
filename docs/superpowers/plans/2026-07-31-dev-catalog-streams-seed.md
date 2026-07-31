# Dev catalog streams seed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a re-runnable `seed-catalog-streams` Mongo CLI (and `task seed:catalog-streams`) that full-replaces Formata stream definitions so every taxonomy leaf has 1–3 categorized, distinctly named catalog streams.

**Architecture:** Mirror `seed-categories`: CLI opens Mongo via `openMongoTaxonomyStore`, builds a minimal `Server` with store + config dir, loads taxonomy leaves and raw YAML templates (Formata `Stream` bodies if any exist, else config-dir workflow files), `DeleteWorkflowData` for current `workflowCatalog()` keys, wipes Formata streams, then inserts clones with category pair + renamed workflow title.

**Tech Stack:** Go `package main` under `server/cmd/server`, Mongo via existing `Store`, `gopkg.in/yaml.v3` / `parseRuntimeConfigData` for validation, Taskfile, `go test` with `MemoryStore`.

**Spec:** `docs/superpowers/specs/2026-07-31-dev-catalog-streams-seed-design.md`

## Global Constraints

- CLI: `go run ./cmd/server seed-catalog-streams`; Taskfile: `task seed:catalog-streams` (dir `server`).
- Mongo: `MONGODB_URI` / `MONGODB_DATABASE` same defaults as categories seed (`mongodb://localhost:27017`, `closer_demo`).
- Full replace of `formata_builder_streams`; random **1–3** streams per taxonomy leaf each run.
- Templates: Formata `Stream` strings when any Formata streams exist; else raw config-dir YAML via `isWorkflowCatalogConfigFile`. Never marshal `RuntimeConfig` back to YAML.
- Before wipe: `DeleteWorkflowData` for every current `workflowCatalog()` key; do not create processes.
- Empty taxonomy or empty templates → error. Empty catalog keys → skip deletes (OK).
- Names: `{SubCategory display name} — {Pilot|Standard|Extended}`.
- Out of scope: on-disk config YAML writes, Appwrite, taxonomy seed, instance seed, auto-run on start/dev.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/catalog_streams_seed.go` | Pure YAML surgery + leaf/target helpers; `seedCatalogStreams` orchestration |
| `server/cmd/server/catalog_streams_seed_test.go` | Unit tests for surgery, counts, replace, process cleanup, file templates |
| `server/cmd/server/catalog_streams_seed_cmd.go` | CLI: open store, config dir, call `seedCatalogStreams`, logging |
| `server/cmd/server/catalog_streams_seed_cmd_test.go` | CLI opener / empty-taxonomy / happy-path with MemoryStore |
| `server/cmd/server/main.go` | Dispatch `seed-catalog-streams` next to other seed subcommands |
| `Taskfile.yml` | `seed:catalog-streams` task |

Reuse (do not duplicate): `openMongoTaxonomyStore` (`taxonomy_seed_cmd.go`), `loadTaxonomyTree`, `isWorkflowCatalogConfigFile`, `workflowCatalog`, `DeleteWorkflowData`, `ListFormataBuilderStreams` / `SaveFormataBuilderStream` / `DeleteFormataBuilderStream`, `parseRuntimeConfigData`, `platformAdminStreamUserID`, `envOr`.

---

### Task 1: YAML surgery + leaf helpers

**Files:**
- Create: `server/cmd/server/catalog_streams_seed.go`
- Create: `server/cmd/server/catalog_streams_seed_test.go`

**Interfaces:**
- Produces:
  - `var catalogStreamNameVariants = []string{"Pilot", "Standard", "Extended"}`
  - `type catalogStreamLeaf struct { CategorySlug, SubCategorySlug, SubCategoryName string }`
  - `func taxonomyLeavesFromTree(tree []TaxonomyCategoryNode) []catalogStreamLeaf` — one entry per sub-category; skip categories with no subs
  - `func catalogStreamDisplayName(subName string, slotIndex int) string` — `fmt.Sprintf("%s — %s", subName, catalogStreamNameVariants[slotIndex%len(...)])`
  - `func applyCatalogStreamCategoryAndName(yamlText, categorySlug, subCategorySlug, name string) (string, error)` — strip all category/subCategory lines (incl. last line without `\n`); insert pair under `workflow:`; set **workflow** `name` only; validate with `parseRuntimeConfigData`; return error if `workflow:` missing or parse fails
  - `func loadCatalogStreamTemplateBodies(ctx context.Context, store Store, configDir string) ([]string, error)` — Formata `Stream` fields if any; else read config-dir workflow files; error if empty
  - `type catalogStreamRNG interface { Intn(n int) int }` — for injectable randomness in later tasks

- [ ] **Step 1: Write failing tests for surgery and naming**

```go
package main

import (
	"strings"
	"testing"
)

func TestCatalogStreamDisplayName(t *testing.T) {
	if got := catalogStreamDisplayName("Procurement", 0); got != "Procurement — Pilot" {
		t.Fatalf("got %q", got)
	}
	if got := catalogStreamDisplayName("Procurement", 2); got != "Procurement — Extended" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyCatalogStreamCategoryAndNameStripsTrailingDupe(t *testing.T) {
	// Formata-shaped: roles before workflow; orphan trailing subCategorySlug without final newline
	in := strings.TrimRight(`roles:
  - orgSlug: org1
    slug: dep1
    name: Department 1
workflow:
  name: Old Name
  categorySlug: supply-chain
  subCategorySlug: procurement
  steps:
    - id: "1"
      title: Step 1
      order: 1
      organization: org1
      substeps:
        - id: "1.1"
          title: Input
          order: 1
          roles: ["dep1"]
          inputKey: value
          inputType: string
organizations:
  - slug: org1
    name: Organization 1
  subCategorySlug: supplier-onboarding`, "\n") // no trailing newline on last key

	out, err := applyCatalogStreamCategoryAndName(in, "recycling-and-recovery", "photovoltaic-panels", "Photovoltaic Panels — Pilot")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Count(out, "categorySlug:") != 1 || strings.Count(out, "subCategorySlug:") != 1 {
		t.Fatalf("duplicate keys remain:\n%s", out)
	}
	if !strings.Contains(out, "categorySlug: recycling-and-recovery") || !strings.Contains(out, "subCategorySlug: photovoltaic-panels") {
		t.Fatalf("missing new pair:\n%s", out)
	}
	if !strings.Contains(out, "name: Photovoltaic Panels — Pilot") {
		t.Fatalf("workflow name not set:\n%s", out)
	}
	if strings.Contains(out, "name: Department 1") == false {
		t.Fatal("role name must be preserved")
	}
	if _, err := parseRuntimeConfigData("clone.yaml", []byte(out)); err != nil {
		t.Fatalf("parseRuntimeConfigData: %v", err)
	}
}

func TestTaxonomyLeavesFromTree(t *testing.T) {
	leaves := taxonomyLeavesFromTree([]TaxonomyCategoryNode{
		{Slug: "a", Name: "A", SubCategories: []TaxonomySubCategoryNode{
			{Slug: "a1", Name: "A1"},
			{Slug: "a2", Name: "A2"},
		}},
		{Slug: "b", Name: "B"}, // no subs
	})
	if len(leaves) != 2 {
		t.Fatalf("len=%d want 2", len(leaves))
	}
	if leaves[0].CategorySlug != "a" || leaves[0].SubCategorySlug != "a1" || leaves[0].SubCategoryName != "A1" {
		t.Fatalf("leaf0=%+v", leaves[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestCatalogStreamDisplayName|TestApplyCatalogStreamCategoryAndNameStripsTrailingDupe|TestTaxonomyLeavesFromTree' -count=1`

Expected: FAIL (undefined symbols)

- [ ] **Step 3: Implement helpers in `catalog_streams_seed.go`**

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var catalogStreamNameVariants = []string{"Pilot", "Standard", "Extended"}

type catalogStreamLeaf struct {
	CategorySlug     string
	SubCategorySlug  string
	SubCategoryName  string
}

type catalogStreamRNG interface {
	Intn(n int) int
}

func taxonomyLeavesFromTree(tree []TaxonomyCategoryNode) []catalogStreamLeaf {
	var out []catalogStreamLeaf
	for _, cat := range tree {
		for _, sub := range cat.SubCategories {
			out = append(out, catalogStreamLeaf{
				CategorySlug:    strings.TrimSpace(cat.Slug),
				SubCategorySlug: strings.TrimSpace(sub.Slug),
				SubCategoryName: strings.TrimSpace(sub.Name),
			})
		}
	}
	return out
}

func catalogStreamDisplayName(subName string, slotIndex int) string {
	variant := catalogStreamNameVariants[slotIndex%len(catalogStreamNameVariants)]
	return fmt.Sprintf("%s — %s", strings.TrimSpace(subName), variant)
}

var (
	reCatalogStreamCat = regexp.MustCompile(`(?m)^\s*categorySlug:.*(?:\n|$)`)
	reCatalogStreamSub = regexp.MustCompile(`(?m)^\s*subCategorySlug:.*(?:\n|$)`)
	reCatalogStreamWF  = regexp.MustCompile(`(?m)^workflow:\n`)
)

func applyCatalogStreamCategoryAndName(yamlText, categorySlug, subCategorySlug, name string) (string, error) {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	name = strings.TrimSpace(name)
	if cat == "" || sub == "" || name == "" {
		return "", fmt.Errorf("categorySlug, subCategorySlug, and name are required")
	}
	out := strings.TrimRight(yamlText, "\n") + "\n"
	out = reCatalogStreamCat.ReplaceAllString(out, "")
	out = reCatalogStreamSub.ReplaceAllString(out, "")
	if !reCatalogStreamWF.MatchString(out) {
		return "", fmt.Errorf("workflow: key not found")
	}
	out = reCatalogStreamWF.ReplaceAllString(out, fmt.Sprintf(
		"workflow:\n  categorySlug: %s\n  subCategorySlug: %s\n", cat, sub,
	))
	replaced := false
	out = regexp.MustCompile(`(?m)^workflow:\n(?:  .+\n)*?  name:\s*.+$`).ReplaceAllStringFunc(out, func(block string) string {
		if replaced {
			return block
		}
		replaced = true
		return regexp.MustCompile(`(?m)^(  name:\s*).+$`).ReplaceAllString(block, "${1}"+name)
	})
	if !replaced {
		return "", fmt.Errorf("workflow name: not found")
	}
	if _, err := parseRuntimeConfigData("catalog-stream-seed.yaml", []byte(out)); err != nil {
		return "", err
	}
	return out, nil
}

func loadCatalogStreamTemplateBodies(ctx context.Context, store Store, configDir string) ([]string, error) {
	if store != nil {
		streams, err := store.ListFormataBuilderStreams(ctx)
		if err != nil {
			return nil, err
		}
		if len(streams) > 0 {
			bodies := make([]string, 0, len(streams))
			for _, s := range streams {
				bodies = append(bodies, s.Stream)
			}
			return bodies, nil
		}
	}
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config dir not found: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isWorkflowCatalogConfigFile(entry.Name()) {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("catalog stream seed: no template workflows found")
	}
	bodies := make([]string, 0, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read template %s: %w", path, readErr)
		}
		bodies = append(bodies, string(data))
	}
	return bodies, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestCatalogStreamDisplayName|TestApplyCatalogStreamCategoryAndNameStripsTrailingDupe|TestTaxonomyLeavesFromTree' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/catalog_streams_seed.go server/cmd/server/catalog_streams_seed_test.go
git commit -m "$(cat <<'EOF'
feat(seed): add catalog stream YAML surgery helpers

Strip/set category pairs and rename workflow titles without
duplicating trailing keys or clobbering role names.
EOF
)"
```

---

### Task 2: `seedCatalogStreams` orchestration

**Files:**
- Modify: `server/cmd/server/catalog_streams_seed.go`
- Modify: `server/cmd/server/catalog_streams_seed_test.go`

**Interfaces:**
- Consumes: helpers from Task 1; `loadTaxonomyTree`; `Server.workflowCatalog`; store Formata + `DeleteWorkflowData`
- Produces:
  - `type catalogStreamsSeedResult struct { Leaves int; Streams int }`
  - `func seedCatalogStreams(ctx context.Context, server *Server, rng catalogStreamRNG) (catalogStreamsSeedResult, error)`
    - requires `server != nil`, `server.store != nil`, `rng != nil`
    - empty leaves / empty templates → error
    - delete workflow data for catalog keys → wipe all Formata → insert clones
    - creator/updater: `platformAdminStreamUserID()`

- [ ] **Step 1: Write failing orchestration tests**

```go
func TestSeedCatalogStreamsFillsEveryLeafAndReplaces(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
			{Slug: "compliance-and-quality", Name: "Compliance", Icon: "quality-control", SortOrder: 2}},
		[]SubCategory{
			{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
			{CategorySlug: "compliance-and-quality", Slug: "inspection", Name: "Inspection", Icon: "inspection-workflow", SortOrder: 1},
		},
	); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	tmpl := minimalCategorizedWorkflowYAML("")
	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: tmpl})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}
	oldProcessID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          oldProcessID,
		WorkflowKey: saved.ID.Hex(),
		Status:      processStatusActive,
		CreatedAt:   time.Now().UTC(),
		Progress:    map[string]ProcessStep{},
	})

	server := &Server{store: store, configDir: t.TempDir()}
	rng := rand.New(rand.NewSource(1))
	res, err := seedCatalogStreams(t.Context(), server, rng)
	if err != nil {
		t.Fatalf("seedCatalogStreams: %v", err)
	}
	if res.Leaves != 2 {
		t.Fatalf("leaves=%d want 2", res.Leaves)
	}
	if res.Streams < 2 || res.Streams > 6 {
		t.Fatalf("streams=%d want 2..6", res.Streams)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(streams) != res.Streams {
		t.Fatalf("listed=%d result=%d", len(streams), res.Streams)
	}
	for _, s := range streams {
		if s.ID == saved.ID {
			t.Fatal("old formata id survived wipe")
		}
	}
	if _, err := store.LoadProcessByID(t.Context(), oldProcessID); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadProcessByID after DeleteWorkflowData: %v, want ErrNoDocuments", err)
	}
	recent, err := store.ListRecentProcessesByWorkflow(t.Context(), saved.ID.Hex(), 10)
	if err != nil {
		t.Fatalf("ListRecentProcessesByWorkflow: %v", err)
	}
	if len(recent) != 0 {
		t.Fatalf("expected no processes for old key, got %d", len(recent))
	}

	// every leaf has 1..3
	counts := map[string]int{}
	for _, s := range streams {
		cfg, err := parseRuntimeConfigData(s.ID.Hex(), []byte(s.Stream))
		if err != nil {
			t.Fatalf("parse %s: %v", s.ID.Hex(), err)
		}
		key := cfg.Workflow.CategorySlug + "/" + cfg.Workflow.SubCategorySlug
		counts[key]++
		if !strings.Contains(cfg.Workflow.Name, " — ") {
			t.Fatalf("name %q missing variant", cfg.Workflow.Name)
		}
	}
	for _, leaf := range []string{"supply-chain/procurement", "compliance-and-quality/inspection"} {
		n := counts[leaf]
		if n < 1 || n > 3 {
			t.Fatalf("%s count=%d", leaf, n)
		}
	}

	// second run replaces
	res2, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	streams2, _ := store.ListFormataBuilderStreams(t.Context())
	if len(streams2) != res2.Streams {
		t.Fatalf("after replace listed=%d", len(streams2))
	}
	ids := map[primitive.ObjectID]bool{}
	for _, s := range streams {
		ids[s.ID] = true
	}
	for _, s := range streams2 {
		if ids[s.ID] {
			t.Fatal("first-run id survived second seed")
		}
	}
}

func TestSeedCatalogStreamsUsesConfigDirWhenFormataEmpty(t *testing.T) {
	store := NewMemoryStore()
	_ = store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1}},
		[]SubCategory{{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1}},
	)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &Server{store: store, configDir: dir}
	res, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res.Streams < 1 {
		t.Fatal("expected at least one stream from file template")
	}
}

func TestSeedCatalogStreamsErrorsWithoutTaxonomy(t *testing.T) {
	store := NewMemoryStore()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: store, configDir: dir}
	if _, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(1))); err == nil {
		t.Fatal("expected error for empty taxonomy")
	}
}
```

**Note for implementer:** Process cleanup asserts via `LoadProcessByID` → `mongo.ErrNoDocuments` and `ListRecentProcessesByWorkflow(oldKey, 10)` empty. Fix icon allowlist values if `ReplaceTaxonomy` rejects icons—copy from `seedSupplyChainProcurementTaxonomy` / `categories.yaml`. Remove the unused `seedSupplyChainProcurementTaxonomy` call if using explicit `ReplaceTaxonomy`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestSeedCatalogStreams' -count=1`

Expected: FAIL (`seedCatalogStreams` undefined)

- [ ] **Step 3: Implement `seedCatalogStreams`**

```go
type catalogStreamsSeedResult struct {
	Leaves  int
	Streams int
}

func seedCatalogStreams(ctx context.Context, server *Server, rng catalogStreamRNG) (catalogStreamsSeedResult, error) {
	var zero catalogStreamsSeedResult
	if server == nil || server.store == nil {
		return zero, fmt.Errorf("catalog stream seed: server store is required")
	}
	if rng == nil {
		return zero, fmt.Errorf("catalog stream seed: rng is required")
	}

	tree, err := loadTaxonomyTree(ctx, server.store)
	if err != nil {
		return zero, err
	}
	leaves := taxonomyLeavesFromTree(tree)
	if len(leaves) == 0 {
		return zero, fmt.Errorf("catalog stream seed: taxonomy is empty (run seed-categories first)")
	}

	templates, err := loadCatalogStreamTemplateBodies(ctx, server.store, server.configDir)
	if err != nil {
		return zero, err
	}

	catalog, err := server.workflowCatalog()
	if err != nil && err.Error() != "workflow config catalog is empty" {
		// If catalog errors only when empty files+formata: treat empty as OK.
		// Prefer: if strings.Contains(err.Error(), "empty") { catalog = nil } else return err
		return zero, err
	}
	for key := range catalog {
		if delErr := server.store.DeleteWorkflowData(ctx, key); delErr != nil {
			return zero, fmt.Errorf("delete workflow data %s: %w", key, delErr)
		}
	}

	existing, err := server.store.ListFormataBuilderStreams(ctx)
	if err != nil {
		return zero, err
	}
	for _, doc := range existing {
		if delErr := server.store.DeleteFormataBuilderStream(ctx, doc.ID); delErr != nil {
			return zero, fmt.Errorf("delete formata stream %s: %w", doc.ID.Hex(), delErr)
		}
	}

	creator := platformAdminStreamUserID()
	now := time.Now().UTC()
	if server.now != nil {
		now = server.now().UTC()
	}
	inserted := 0
	for _, leaf := range leaves {
		n := 1 + rng.Intn(3)
		for i := 0; i < n; i++ {
			body := templates[rng.Intn(len(templates))]
			name := catalogStreamDisplayName(leaf.SubCategoryName, i)
			yamlOut, applyErr := applyCatalogStreamCategoryAndName(body, leaf.CategorySlug, leaf.SubCategorySlug, name)
			if applyErr != nil {
				return zero, fmt.Errorf("build %s/%s: %w", leaf.CategorySlug, leaf.SubCategorySlug, applyErr)
			}
			if _, saveErr := server.store.SaveFormataBuilderStream(ctx, FormataBuilderStream{
				Stream:          yamlOut,
				UpdatedAt:       now,
				CreatedByUserID: creator,
				UpdatedByUserID: creator,
			}); saveErr != nil {
				return zero, fmt.Errorf("save %s/%s: %w", leaf.CategorySlug, leaf.SubCategorySlug, saveErr)
			}
			inserted++
		}
	}
	return catalogStreamsSeedResult{Leaves: len(leaves), Streams: inserted}, nil
}
```

Handle empty-catalog error the same way other code does (grep `workflow config catalog is empty` in `public_home.go` / instance seed). Do not fail the seed when the catalog is empty before wipe.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestSeedCatalogStreams|TestApplyCatalogStream|TestCatalogStreamDisplay|TestTaxonomyLeaves' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/catalog_streams_seed.go server/cmd/server/catalog_streams_seed_test.go
git commit -m "$(cat <<'EOF'
feat(seed): orchestrate full-replace categorized catalog streams

Wipe Formata streams after deleting old workflow data, then insert
1–3 clones per taxonomy leaf from catalog templates.
EOF
)"
```

---

### Task 3: CLI, main dispatch, Taskfile

**Files:**
- Create: `server/cmd/server/catalog_streams_seed_cmd.go`
- Create: `server/cmd/server/catalog_streams_seed_cmd_test.go`
- Modify: `server/cmd/server/main.go` (seed dispatch near `seed-categories`; if `seed-instances` already exists, add beside it—prefer a small `switch` only if that file already uses one)
- Modify: `Taskfile.yml` (add `seed:catalog-streams` beside `seed:categories`)

**Interfaces:**
- Consumes: `seedCatalogStreams`, `openMongoTaxonomyStore`
- Produces: `func runSeedCatalogStreamsCommand(ctx context.Context, args []string) error` — ignores args in v1; opens store; builds `Server{store, configDir}`; `rng := rand.New(rand.NewSource(time.Now().UnixNano()))`; logs result + reminder to run `seed-instances`

- [ ] **Step 1: Write failing CLI test**

```go
func TestRunSeedCatalogStreamsCommandWithMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	_ = store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1}},
		[]SubCategory{{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1}},
	)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)

	err := seedCatalogStreamsWithStoreOpener(t.Context(), nil, dir, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	})
	if err != nil {
		t.Fatalf("cmd: %v", err)
	}
	streams, _ := store.ListFormataBuilderStreams(t.Context())
	if len(streams) < 1 || len(streams) > 3 {
		t.Fatalf("streams=%d", len(streams))
	}
}

func TestRunSeedCatalogStreamsCommandRequiresStore(t *testing.T) {
	err := seedCatalogStreamsWithStoreOpener(t.Context(), nil, t.TempDir(), func(context.Context) (Store, func(), error) {
		return nil, nil, fmt.Errorf("boom")
	})
	if err == nil {
		t.Fatal("expected opener error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server -run 'TestRunSeedCatalogStreamsCommand' -count=1`

Expected: FAIL (undefined)

- [ ] **Step 3: Implement CLI + wiring**

`catalog_streams_seed_cmd.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runSeedCatalogStreamsCommand(ctx context.Context, args []string) error {
	_ = args
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return seedCatalogStreamsWithStoreOpener(ctx, nil, configDir, openMongoTaxonomyStore)
}

func seedCatalogStreamsWithStoreOpener(
	ctx context.Context,
	rng catalogStreamRNG,
	configDir string,
	open func(context.Context) (Store, func(), error),
) error {
	if open == nil {
		return fmt.Errorf("catalog stream seed: store opener is required")
	}
	store, cleanup, err := open(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	server := &Server{store: store, configDir: configDir}
	res, err := seedCatalogStreams(ctx, server, rng)
	if err != nil {
		return err
	}
	log.Printf("seeded %d catalog streams across %d taxonomy leaves; next: task seed:instances", res.Streams, res.Leaves)
	return nil
}
```

In `main.go`, next to the existing `seed-categories` block (and `seed-instances` if present):

```go
if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "seed-catalog-streams" {
	if err := runSeedCatalogStreamsCommand(ctx, os.Args[2:]); err != nil {
		log.Fatal(err)
	}
	return
}
```

In `Taskfile.yml` after `seed:categories`:

```yaml
  seed:catalog-streams:
    desc: Full-replace Formata catalog streams (1–3 per taxonomy leaf) for local/dev.
    dir: server
    cmds:
      - go run ./cmd/server seed-catalog-streams {{.CLI_ARGS}}
```

- [ ] **Step 4: Run tests**

Run: `cd server && go test ./cmd/server -run 'TestRunSeedCatalogStreamsCommand|TestSeedCatalogStreams|TestApplyCatalogStream' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/catalog_streams_seed_cmd.go server/cmd/server/catalog_streams_seed_cmd_test.go server/cmd/server/main.go Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(seed): add seed-catalog-streams CLI and task

Wire Mongo opener + Taskfile so demo catalogs can be rebuilt
before seed-instances.
EOF
)"
```

- [ ] **Step 6: Manual smoke (optional if stack is up)**

```bash
task seed:categories   # if needed
task seed:catalog-streams
# then, when available: task seed:instances
```

Expected: log line with leaf/stream counts; `/` subcategory filters show cards; no YAML duplicate-key errors in server logs.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `seed-catalog-streams` + `task seed:catalog-streams` | 3 |
| Full Formata wipe + insert | 2 |
| Random 1–3 per leaf | 2 |
| Templates Formata-else-files | 1 (`loadCatalogStreamTemplateBodies`) + 2 |
| `DeleteWorkflowData` on old catalog keys | 2 |
| Taxonomy empty → error | 2 |
| Naming Pilot/Standard/Extended | 1 |
| No duplicate category keys / preserve role names | 1 |
| No on-disk YAML / no instances created | 2–3 (out of scope enforced) |
| Unit tests listed in spec | 1–2 |
| Run order docs in log reminder | 3 |

## Placeholder / consistency self-review

- No TBD steps; interfaces name `seedCatalogStreams` / `applyCatalogStreamCategoryAndName` consistently.
- RNG injected via `catalogStreamRNG` for tests.
- Empty-catalog error handling called out as implementer must match existing string/`errors.Is` pattern.
- Process cleanup uses `LoadProcessByID` + `ListRecentProcessesByWorkflow` (confirmed on `Store`).
