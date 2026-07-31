# Dev stream instance seed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a re-runnable `seed-instances` Mongo CLI (and `task seed:instances`) that replaces each runtime-catalog stream’s processes with a fixed mix of realistic active/done/terminated instances so public and `/my` UIs look populated.

**Architecture:** Mirror `seed-categories`: CLI subcommand opens Mongo via the same store opener pattern, loads the catalog through a minimal `Server` using `workflowCatalog()`, then per key calls `DeleteWorkflowData` and inserts algorithmically generated processes (progress from `orderedSubsteps`, real GridFS via `SaveAttachment` for `file` steps, DPP via `buildProcessDPP` + `UpdateProcessDPP`).

**Tech Stack:** Go `package main` under `server/cmd/server`, MongoDB + GridFS via existing `Store`, Taskfile, `go test` with `MemoryStore`.

**Spec:** `docs/superpowers/specs/2026-07-31-dev-instance-seed-design.md`

## Global Constraints

- CLI: `go run ./cmd/server seed-instances`; Taskfile: `task seed:instances` (dir `server`).
- Mongo: `MONGODB_URI` / `MONGODB_DATABASE` same defaults as categories seed (`mongodb://localhost:27017`, `closer_demo`).
- Catalog: exactly `workflowCatalog()` rules (Formata-only when any Formata streams exist; else YAML config dir). No second merge.
- Replace per key: `DeleteWorkflowData(key)` then insert; do not touch taxonomy, Appwrite, or Formata definitions.
- Mix per stream: 1 fresh active, 2 mid active (~40% progress), 3 done (2 with DPP when enabled, 1 without), 2 terminated → **8** total.
- File substeps: real `SaveAttachment` blobs; never fake `attachmentId`.
- Abort on first per-stream failure (delete/insert/DPP/file); log failing key; non-zero exit.
- Out of scope: auto-run on start/dev, HTTP mocks, YAML process fixtures.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/instances_seed.go` | Pure plan builder + progress/Data synthesis; `seedWorkflowInstances` persist orchestration |
| `server/cmd/server/instances_seed_test.go` | MemoryStore unit tests for mix, replace, DPP, files |
| `server/cmd/server/instances_seed_cmd.go` | CLI: open store, load catalog, loop keys, logging |
| `server/cmd/server/instances_seed_cmd_test.go` | CLI opener / empty-catalog / happy-path with MemoryStore |
| `server/cmd/server/main.go` | Dispatch `seed-instances` next to `seed-categories` |
| `Taskfile.yml` | `seed:instances` task |

Reuse (do not duplicate): `openMongoTaxonomyStore` from `taxonomy_seed_cmd.go` (same Mongo opener), `orderedSubsteps`, `encodeProgressKey`, `buildProcessDPP`, `DeleteWorkflowData`, `InsertProcess`, `SaveAttachment`, `UpdateProcessDPP`, `processStatus*`, `workflowCatalog`.

---

### Task 1: Pure instance plan + progress synthesis

**Files:**
- Create: `server/cmd/server/instances_seed.go`
- Create: `server/cmd/server/instances_seed_test.go`

**Interfaces:**
- Produces:
  - `type seedInstanceKind string` with constants `seedKindFreshActive`, `seedKindMidActive`, `seedKindDoneWithDPP`, `seedKindDoneNoDPP`, `seedKindTerminated`
  - `type seedInstancePlan struct { Kind seedInstanceKind; Name string; Status string; DoneSubstepCount int; TerminationReason string; WantDPP bool; CreatedOffset time.Duration }`
  - `func buildSeedInstancePlans(cfg RuntimeConfig, now time.Time) []seedInstancePlan` — always length 8 in fixed order
  - `func seedDoneSubstepCount(total int, fraction float64) int` — mid ≈ 40%
  - `func synthesizeSeedProgress(def WorkflowDef, doneCount int, now time.Time, actor Actor) map[string]ProcessStep` — keys **encoded** via `encodeProgressKey`; only first `doneCount` substeps present as `state: done` with `DoneAt`/`DoneBy`/`Data` (non-file Data only; file steps get empty `Data` placeholder filled later)
  - `func synthesizeSeedStepData(sub WorkflowSub, cfg RuntimeConfig) map[string]interface{}` — scalars / formata object; for `file` return `nil` (materializer fills); when `cfg.DPP.Enabled` and substep can supply `LotInputKey`, include that lot string (e.g. `"SEED-LOT-001"`)

- [ ] **Step 1: Write failing tests for plan mix and progress**

```go
func TestBuildSeedInstancePlansMix(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name: "Demo",
			Steps: []WorkflowStep{{
				StepID: "1", Order: 1, Title: "S1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Order: 1, Title: "A", InputKey: "a", InputType: "string"},
					{SubstepID: "1.2", Order: 2, Title: "B", InputKey: "b", InputType: "string"},
					{SubstepID: "1.3", Order: 3, Title: "C", InputKey: "c", InputType: "string"},
					{SubstepID: "1.4", Order: 4, Title: "D", InputKey: "d", InputType: "string"},
					{SubstepID: "1.5", Order: 5, Title: "E", InputKey: "e", InputType: "string"},
				},
			}},
		},
		DPP: DPPConfig{Enabled: true, GTIN: "09506000134352", LotDefault: "LOT", SerialStrategy: "process_id_hex"},
	}
	plans := buildSeedInstancePlans(cfg, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC))
	if len(plans) != 8 {
		t.Fatalf("len(plans) = %d, want 8", len(plans))
	}
	var fresh, mid, doneDPP, doneNo, term int
	for _, p := range plans {
		switch p.Kind {
		case seedKindFreshActive:
			fresh++
			if p.Status != processStatusActive || p.DoneSubstepCount != 0 {
				t.Fatalf("fresh plan = %+v", p)
			}
		case seedKindMidActive:
			mid++
			if p.Status != processStatusActive || p.DoneSubstepCount != 2 { // 40% of 5 → 2
				t.Fatalf("mid plan = %+v", p)
			}
		case seedKindDoneWithDPP:
			doneDPP++
			if p.Status != processStatusDone || !p.WantDPP || p.DoneSubstepCount != 5 {
				t.Fatalf("done+dpp = %+v", p)
			}
		case seedKindDoneNoDPP:
			doneNo++
			if p.Status != processStatusDone || p.WantDPP || p.DoneSubstepCount != 5 {
				t.Fatalf("done-no-dpp = %+v", p)
			}
		case seedKindTerminated:
			term++
			if p.Status != processStatusTerminated || p.TerminationReason == "" {
				t.Fatalf("terminated = %+v", p)
			}
		default:
			t.Fatalf("unknown kind %q", p.Kind)
		}
	}
	if fresh != 1 || mid != 2 || doneDPP != 2 || doneNo != 1 || term != 2 {
		t.Fatalf("counts fresh=%d mid=%d doneDPP=%d doneNo=%d term=%d", fresh, mid, doneDPP, doneNo, term)
	}
}

func TestBuildSeedInstancePlansNoDPPWhenDisabled(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{Name: "X", Steps: []WorkflowStep{{
			StepID: "1", Order: 1,
			Substep: []WorkflowSub{{SubstepID: "1.1", Order: 1, InputKey: "a", InputType: "string"}},
		}}},
		DPP: DPPConfig{Enabled: false},
	}
	plans := buildSeedInstancePlans(cfg, time.Now().UTC())
	for _, p := range plans {
		if p.Kind == seedKindDoneWithDPP || p.WantDPP {
			t.Fatalf("DPP disabled but plan wants DPP: %+v", p)
		}
	}
	var done int
	for _, p := range plans {
		if p.Status == processStatusDone {
			done++
		}
	}
	if done != 3 {
		t.Fatalf("done count = %d, want 3", done)
	}
}

func TestSynthesizeSeedProgressEncodedKeysAndPrefix(t *testing.T) {
	def := WorkflowDef{Steps: []WorkflowStep{{
		StepID: "1", Order: 1,
		Substep: []WorkflowSub{
			{SubstepID: "1.1", Order: 1, InputKey: "a", InputType: "string"},
			{SubstepID: "1.2", Order: 2, InputKey: "b", InputType: "string"},
			{SubstepID: "1.3", Order: 3, InputKey: "c", InputType: "string"},
		},
	}}}
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	actor := Actor{ID: "seed", Role: "seed"}
	progress := synthesizeSeedProgress(def, 2, now, actor)
	if _, ok := progress["1_1"]; !ok {
		t.Fatalf("expected encoded key 1_1, got %#v", progress)
	}
	if _, ok := progress["1.1"]; ok {
		t.Fatal("must not store dotted key 1.1")
	}
	if len(progress) != 2 {
		t.Fatalf("len(progress) = %d, want 2", len(progress))
	}
	if progress["1_2"].State != "done" || progress["1_2"].DoneBy == nil {
		t.Fatalf("1_2 = %+v", progress["1_2"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestBuildSeedInstancePlans|TestSynthesizeSeedProgress' -count=1`

Expected: FAIL — undefined `buildSeedInstancePlans` / `synthesizeSeedProgress`.

- [ ] **Step 3: Implement plan + progress helpers in `instances_seed.go`**

```go
const (
	seedKindFreshActive  seedInstanceKind = "fresh_active"
	seedKindMidActive    seedInstanceKind = "mid_active"
	seedKindDoneWithDPP  seedInstanceKind = "done_with_dpp"
	seedKindDoneNoDPP    seedInstanceKind = "done_no_dpp"
	seedKindTerminated   seedInstanceKind = "terminated"
	seedLotValue         = "SEED-LOT-001"
)

func seedDoneSubstepCount(total int, fraction float64) int {
	if total <= 0 {
		return 0
	}
	n := int(float64(total) * fraction)
	if n < 1 && total >= 1 && fraction > 0 {
		n = 1
	}
	if n > total {
		n = total
	}
	return n
}

func buildSeedInstancePlans(cfg RuntimeConfig, now time.Time) []seedInstancePlan {
	_ = now
	total := len(orderedSubsteps(cfg.Workflow))
	midDone := seedDoneSubstepCount(total, 0.4)
	termDone := seedDoneSubstepCount(total, 0.5)
	name := strings.TrimSpace(cfg.Workflow.Name)
	if name == "" {
		name = "Stream"
	}
	dppOn := cfg.DPP.Enabled
	plans := []seedInstancePlan{
		{Kind: seedKindFreshActive, Name: name + " — just started", Status: processStatusActive, DoneSubstepCount: 0, CreatedOffset: -7 * time.Minute},
		{Kind: seedKindMidActive, Name: name + " — mid run A", Status: processStatusActive, DoneSubstepCount: midDone, CreatedOffset: -6 * time.Minute},
		{Kind: seedKindMidActive, Name: name + " — mid run B", Status: processStatusActive, DoneSubstepCount: midDone, CreatedOffset: -5 * time.Minute},
	}
	if dppOn {
		plans = append(plans,
			seedInstancePlan{Kind: seedKindDoneWithDPP, Name: name + " — completed (passport)", Status: processStatusDone, DoneSubstepCount: total, WantDPP: true, CreatedOffset: -4 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneWithDPP, Name: name + " — completed (passport 2)", Status: processStatusDone, DoneSubstepCount: total, WantDPP: true, CreatedOffset: -3 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed", Status: processStatusDone, DoneSubstepCount: total, WantDPP: false, CreatedOffset: -2 * time.Minute},
		)
	} else {
		plans = append(plans,
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed A", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -4 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed B", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -3 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed C", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -2 * time.Minute},
		)
	}
	plans = append(plans,
		seedInstancePlan{Kind: seedKindTerminated, Name: name + " — stopped", Status: processStatusTerminated, DoneSubstepCount: termDone, TerminationReason: "Seeded termination: operator stopped run", CreatedOffset: -time.Minute},
		seedInstancePlan{Kind: seedKindTerminated, Name: name + " — abandoned", Status: processStatusTerminated, DoneSubstepCount: termDone, TerminationReason: "Seeded termination: abandoned incomplete run", CreatedOffset: -30 * time.Second},
	)
	return plans
}
```

Implement `synthesizeSeedStepData` / `synthesizeSeedProgress` to match tests: encoded keys; for `string`/`number` put a simple value under `InputKey` or `"value"`; for `formata`/object put a small map including required-looking fields; if `cfg.DPP.Enabled` and (`InputKey == LotInputKey` or schema property equals lot key), set lot to `seedLotValue`. For `file`, set `State: done` with `Data: nil` (filled in Task 2).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestBuildSeedInstancePlans|TestSynthesizeSeedProgress' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/instances_seed.go server/cmd/server/instances_seed_test.go
git commit -m "$(cat <<'EOF'
feat(seed): add instance plan and progress synthesis helpers

EOF
)"
```

---

### Task 2: Persist `seedWorkflowInstances` (replace, files, DPP)

**Files:**
- Modify: `server/cmd/server/instances_seed.go`
- Modify: `server/cmd/server/instances_seed_test.go`

**Interfaces:**
- Consumes: `buildSeedInstancePlans`, `synthesizeSeedProgress`, `synthesizeSeedStepData`, `buildProcessDPP`
- Produces:
  - `func seedActor() Actor` → `{ID: "seed-user", Role: "seed", OrgSlug: "seed"}`
  - `func seedWorkflowInstances(ctx context.Context, store Store, workflowKey string, cfg RuntimeConfig, now time.Time) (int, error)`  
    Returns inserted count. Steps: `DeleteWorkflowData` → for each plan materialize process (pre-allocate `ID`, fill file attachments via `SaveAttachment` when `inputType == "file"` for done substeps, `InsertProcess`, if `WantDPP` then `buildProcessDPP` + `UpdateProcessDPP`). On any error wrap with workflow key.

**Attachment Data shape** (must match completion):

```go
map[string]interface{}{
  "attachmentId": attachment.ID.Hex(),
  "filename":     attachment.Filename,
  "contentType":  attachment.ContentType,
  "size":         attachment.SizeBytes,
  "sha256":       attachment.SHA256,
}
```

Upload content: `bytes.NewReader([]byte("attesta seed attachment\n"))`, filename `seed-{substepID}.txt`, content type `text/plain`.

For terminated plans set:

```go
Termination: &ProcessTermination{
  Reason:  plan.TerminationReason,
  EndedAt: now.Add(plan.CreatedOffset).Add(30 * time.Second),
  Actor:   &actor,
}
```

When `WantDPP` is true and `buildProcessDPP` fails → return error (do not leave done-without-passport for that slot). Ensure lot is present in progress when `LotDefault` is empty (Task 1 data synthesis + `seedLotValue`).

- [ ] **Step 1: Write failing persist tests**

```go
func testSeedWorkflowConfig(dpp bool, withFile bool) RuntimeConfig {
	subs := []WorkflowSub{
		{SubstepID: "1.1", Order: 1, Title: "Text", InputKey: "note", InputType: "string"},
		{SubstepID: "1.2", Order: 2, Title: "Batch", InputKey: "batchId", InputType: "formata",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"batchId": map[string]interface{}{"type": "string"},
				},
			},
		},
	}
	if withFile {
		subs = append(subs, WorkflowSub{SubstepID: "1.3", Order: 3, Title: "Doc", InputKey: "doc", InputType: "file"})
	}
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{Name: "Seed WF", Steps: []WorkflowStep{{StepID: "1", Order: 1, Substep: subs}}},
	}
	if dpp {
		cfg.DPP = DPPConfig{Enabled: true, GTIN: "09506000134352", LotInputKey: "batchId", SerialStrategy: "process_id_hex"}
	}
	return cfg
}

func TestSeedWorkflowInstancesReplaceAndMix(t *testing.T) {
	store := NewMemoryStore()
	cfg := testSeedWorkflowConfig(true, false)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	n, err := seedWorkflowInstances(t.Context(), store, "wf-seed", cfg, now)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 8 {
		t.Fatalf("inserted = %d, want 8", n)
	}
	// leave junk then re-seed
	store.SeedProcess(Process{WorkflowKey: "wf-seed", Status: processStatusActive, CreatedAt: now})
	n, err = seedWorkflowInstances(t.Context(), store, "wf-seed", cfg, now)
	if err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-seed", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 8 {
		t.Fatalf("after replace len = %d, want 8", len(list))
	}
	var active, done, term, withDPP int
	for _, p := range list {
		switch p.Status {
		case processStatusActive:
			active++
		case processStatusDone:
			done++
			if p.DPP != nil {
				withDPP++
			}
		case processStatusTerminated:
			term++
			if p.Termination == nil || p.Termination.Reason == "" {
				t.Fatalf("terminated missing reason: %+v", p)
			}
		}
	}
	if active != 3 || done != 3 || term != 2 {
		t.Fatalf("status counts active=%d done=%d term=%d", active, done, term)
	}
	if withDPP != 2 {
		t.Fatalf("withDPP = %d, want 2", withDPP)
	}
}

func TestSeedWorkflowInstancesFileAttachmentRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	cfg := testSeedWorkflowConfig(false, true)
	now := time.Now().UTC()
	if _, err := seedWorkflowInstances(t.Context(), store, "wf-file", cfg, now); err != nil {
		t.Fatalf("seed: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-file", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, p := range list {
		if p.Status != processStatusDone {
			continue
		}
		progress := normalizeProgressKeys(p.Progress)
		step, ok := progress["1.3"]
		if !ok || step.State != "done" || step.Data == nil {
			continue
		}
		id, _ := step.Data["attachmentId"].(string)
		if id == "" {
			t.Fatalf("missing attachmentId in %#v", step.Data)
		}
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			t.Fatalf("attachment id: %v", err)
		}
		if _, err := store.LoadAttachmentByID(t.Context(), oid); err != nil {
			t.Fatalf("LoadAttachmentByID: %v", err)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("expected a done process with file progress")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestSeedWorkflowInstances' -count=1`

Expected: FAIL — `seedWorkflowInstances` undefined.

- [ ] **Step 3: Implement `seedWorkflowInstances`**

Pseudocode:

```go
func seedWorkflowInstances(ctx context.Context, store Store, workflowKey string, cfg RuntimeConfig, now time.Time) (int, error) {
	key := strings.TrimSpace(workflowKey)
	if key == "" {
		return 0, fmt.Errorf("seed instances: workflow key is required")
	}
	if store == nil {
		return 0, fmt.Errorf("seed instances: store is required")
	}
	if err := store.DeleteWorkflowData(ctx, key); err != nil {
		return 0, fmt.Errorf("seed instances %s: delete: %w", key, err)
	}
	actor := seedActor()
	plans := buildSeedInstancePlans(cfg, now)
	inserted := 0
	for _, plan := range plans {
		id := primitive.NewObjectID()
		created := now.Add(plan.CreatedOffset)
		progress := synthesizeSeedProgress(cfg.Workflow, plan.DoneSubstepCount, created, actor)
		// fill file Data for done file substeps
		subs := orderedSubsteps(cfg.Workflow)
		for i := 0; i < plan.DoneSubstepCount && i < len(subs); i++ {
			sub := subs[i]
			if strings.TrimSpace(sub.InputType) != "file" {
				continue
			}
			att, err := store.SaveAttachment(ctx, AttachmentUpload{
				ProcessID: id, SubstepID: sub.SubstepID,
				Filename: "seed-" + sub.SubstepID + ".txt", ContentType: "text/plain",
				UploadedAt: created,
			}, bytes.NewReader([]byte("attesta seed attachment\n")))
			if err != nil {
				return inserted, fmt.Errorf("seed instances %s: attachment %s: %w", key, sub.SubstepID, err)
			}
			enc := encodeProgressKey(sub.SubstepID)
			step := progress[enc]
			step.Data = map[string]interface{}{
				"attachmentId": att.ID.Hex(),
				"filename":     att.Filename,
				"contentType":  att.ContentType,
				"size":         att.SizeBytes,
				"sha256":       att.SHA256,
			}
			progress[enc] = step
		}
		proc := Process{
			ID: id, WorkflowKey: key, Name: plan.Name,
			CreatedAt: created, CreatedBy: actor.ID,
			Status: plan.Status, Progress: progress,
		}
		if plan.Status == processStatusTerminated {
			ended := created.Add(30 * time.Second)
			a := actor
			proc.Termination = &ProcessTermination{Reason: plan.TerminationReason, EndedAt: ended, Actor: &a}
		}
		if _, err := store.InsertProcess(ctx, proc); err != nil {
			return inserted, fmt.Errorf("seed instances %s: insert: %w", key, err)
		}
		inserted++
		if plan.WantDPP {
			dpp, err := buildProcessDPP(cfg.Workflow, cfg.DPP, &proc, created)
			if err != nil {
				return inserted, fmt.Errorf("seed instances %s: dpp: %w", key, err)
			}
			if err := store.UpdateProcessDPP(ctx, id, key, dpp); err != nil {
				return inserted, fmt.Errorf("seed instances %s: update dpp: %w", key, err)
			}
		}
	}
	return inserted, nil
}
```

Note: `buildProcessDPP` reads progress with **decoded** substep IDs via `orderedSubsteps` + `process.Progress[substep.SubstepID]`. After insert, either pass a copy with `normalizeProgressKeys(progress)` into `buildProcessDPP`, or normalize on `proc` before calling it:

```go
proc.Progress = normalizeProgressKeys(proc.Progress)
dpp, err := buildProcessDPP(cfg.Workflow, cfg.DPP, &proc, created)
```

Keep Mongo stored progress encoded (re-encode before insert if you normalized a working copy). Safest: keep `progress` encoded for `InsertProcess`; for DPP build a clone:

```go
clone := proc
clone.Progress = normalizeProgressKeys(cloneProcess(proc).Progress)
dpp, err := buildProcessDPP(cfg.Workflow, cfg.DPP, &clone, created)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server -run 'TestSeedWorkflowInstances|TestBuildSeedInstancePlans|TestSynthesizeSeedProgress' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/instances_seed.go server/cmd/server/instances_seed_test.go
git commit -m "$(cat <<'EOF'
feat(seed): persist realistic stream instances into the store

EOF
)"
```

---

### Task 3: CLI command, main dispatch, Taskfile

**Files:**
- Create: `server/cmd/server/instances_seed_cmd.go`
- Create: `server/cmd/server/instances_seed_cmd_test.go`
- Modify: `server/cmd/server/main.go` (dispatch next to `seed-categories`, ~lines 693–698)
- Modify: `Taskfile.yml` (add `seed:instances` beside `seed:categories`)

**Interfaces:**
- Produces:
  - `func runSeedInstancesCommand(ctx context.Context, args []string) error`
  - `func seedInstancesWithStoreOpener(ctx context.Context, args []string, open func(context.Context) (Store, func(), error)) error` — ignore `args` in v1 (reserved)
  - Uses `openMongoTaxonomyStore` as default opener
  - Builds `&Server{store: store, configDir: …}` where `configDir` comes from `WORKFLOW_CONFIG_DIR` or `filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))`
  - `catalog, err := server.workflowCatalog()`; if `len(catalog)==0` → error
  - Stable key order: sort keys alphabetically
  - Sum inserted counts; `log.Printf("seeded %d instances across %d streams", total, len(keys))`

- [ ] **Step 1: Write failing CLI tests**

```go
func TestSeedInstancesWithStoreOpenerSeedsCatalog(t *testing.T) {
	dir := t.TempDir()
	writeMinimalWorkflowYAML(t, filepath.Join(dir, "demo.yaml"), "Demo Stream")
	store := NewMemoryStore()
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir) // see note below if signature includes configDir
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
	dir := t.TempDir() // no yaml
	store := NewMemoryStore()
	err := seedInstancesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, dir)
	if err == nil {
		t.Fatal("expected empty catalog error")
	}
}
```

Prefer signature:

```go
func seedInstancesWithStoreOpener(ctx context.Context, args []string, open func(context.Context) (Store, func(), error), configDir string) error
```

`runSeedInstancesCommand` resolves `configDir` from env then calls opener + `seedInstancesWithStoreOpener`.

Helper `writeMinimalWorkflowYAML` — small workflow with ≥5 string substeps so mid-progress is meaningful (or reuse patterns from `home_handler_test.go` / `writePublicHomeWorkflowConfig` if already exported in `_test.go`; if unexported, duplicate a 10-line writer in `instances_seed_cmd_test.go`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server -run 'TestSeedInstancesWithStoreOpener' -count=1`

Expected: FAIL — undefined.

- [ ] **Step 3: Implement CLI + wire main + Taskfile**

`instances_seed_cmd.go`:

```go
func runSeedInstancesCommand(ctx context.Context, args []string) error {
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(envOr("WORKFLOW_CONFIG", "config/workflow.yaml"))
	}
	return seedInstancesWithStoreOpener(ctx, args, openMongoTaxonomyStore, configDir)
}

func seedInstancesWithStoreOpener(ctx context.Context, _ []string, open func(context.Context) (Store, func(), error), configDir string) error {
	if open == nil {
		return fmt.Errorf("instance seed: store opener is required")
	}
	store, cleanup, err := open(ctx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	server := &Server{store: store, configDir: configDir, now: func() time.Time { return time.Now().UTC() }}
	catalog, err := server.workflowCatalog()
	if err != nil {
		return fmt.Errorf("instance seed: catalog: %w", err)
	}
	if len(catalog) == 0 {
		return fmt.Errorf("instance seed: workflow catalog is empty")
	}
	keys := make([]string, 0, len(catalog))
	for k := range catalog {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	now := time.Now().UTC()
	if server.now != nil {
		now = server.now()
	}
	total := 0
	for _, key := range keys {
		n, err := seedWorkflowInstances(ctx, store, key, catalog[key], now)
		if err != nil {
			return err
		}
		total += n
	}
	log.Printf("seeded %d instances across %d streams", total, len(keys))
	return nil
}
```

In `main.go` immediately after the `seed-categories` block:

```go
if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "seed-instances" {
	if err := runSeedInstancesCommand(ctx, os.Args[2:]); err != nil {
		log.Fatal(err)
	}
	return
}
```

Taskfile:

```yaml
  seed:instances:
    desc: Replace Mongo process instances for every runtime-catalog stream with a realistic demo mix.
    dir: server
    cmds:
      - go run ./cmd/server seed-instances {{.CLI_ARGS}}
```

- [ ] **Step 4: Run unit tests**

Run: `cd server && go test ./cmd/server -run 'TestSeedInstancesWithStoreOpener|TestSeedWorkflowInstances|TestBuildSeedInstancePlans' -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/instances_seed_cmd.go server/cmd/server/instances_seed_cmd_test.go server/cmd/server/main.go Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(seed): add seed-instances CLI and task seed:instances

EOF
)"
```

---

### Task 4: Manual verification checklist (no code)

**Files:** none

- [ ] **Step 1: Ensure stack + server config**

Run infra (`task start` or `task dev`). Confirm Mongo reachable and `server/config/` workflows load (or Formata streams if any).

- [ ] **Step 2: Seed**

Run: `task seed:instances`

Expected log line like: `seeded N instances across M streams` (N = 8×M).

- [ ] **Step 3: Spot-check UIs**

1. `GET /` — stream cards show non-zero run metrics / active-now where seeded active exists.
2. `GET /streams/<key>` — “Recent completed runs” non-empty; DPP links when passport enabled.
3. `GET /my/streams/<key>/` (authenticated) — active / done / terminated groups populated.
4. Open one `done` instance; timeline shows completed substeps.
5. Re-run `task seed:instances` — counts stay at 8 per stream (replace), no duplicates piling up.

- [ ] **Step 4: Commit nothing** unless you fixed bugs found in verification (then commit those fixes separately).

---

## Spec coverage (self-review)

| Spec requirement | Task |
|------------------|------|
| Mongo CLI like seed-categories | Task 3 |
| `task seed:instances` | Task 3 |
| Full runtime catalog via `workflowCatalog()` | Task 3 |
| Replace via `DeleteWorkflowData` | Task 2 |
| 8-instance mix / statuses | Task 1–2 |
| Realistic progress + termination | Task 1–2 |
| DPP on some done when enabled | Task 1–2 |
| Real GridFS for file steps | Task 2 |
| Abort on failure / logging | Task 2–3 |
| Unit tests MemoryStore | Task 1–3 |
| Manual verify public + /my | Task 4 |
| No auto-start / no mocks / no taxonomy touch | Global constraints |

No TBD placeholders. DPP lot uses `seedLotValue` / `LotInputKey` so `buildProcessDPP` succeeds when `lotDefault` is empty (as in `workflow.yaml`).
