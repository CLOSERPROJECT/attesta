# Timeline Substep Shell Body-First Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `substep_shell` read display chrome (status, palette, done meta) exclusively from `TimelineSubstep.Body`, eliminating duplicate summary fields on the shell path.

**Architecture:** `buildTimelineSubstep` attaches a minimal `SubstepBodyView` shell stub (status/palette/done fields only). `decorateTimelineSubstepBodies` replaces stubs with full bodies from `buildSubstepViews`. `substepShellDisplay` reads only `Body`, with a narrow nil-body fallback for structural tests. DPP builder stops duplicating shell fields on `TimelineSubstep` summary. Coordinate with parallel a11y work: do not change `aria-*` attributes in templates unless tests break.

**Tech Stack:** Go 1.25 (`server/cmd/server`), `html/template`, existing test harness (`parseTestTemplates`, `MemoryStore`).

**Worktree branch:** `feature/timeline-substep-shell-body-first`

---

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/timeline_builder.go` | Attach minimal shell `Body` in `buildTimelineSubstep` |
| `server/cmd/server/components.go` | Simplify `substepShellDisplay`; update `TimelineSubstep` comment |
| `server/cmd/server/dpp.go` | Stop duplicating shell fields on `TimelineSubstep` when `Body` is set |
| `server/cmd/server/timeline_builder_test.go` | Assert minimal body attached; shell display tests |
| `server/cmd/server/substep_shell_test.go` | Update fixtures to body-first |
| `server/cmd/server/stream_timeline_test.go` | Update fixtures to body-first |

**Do not modify:** `server/templates/components/substep_shell.html` except if compile/render breaks (parallel a11y track owns template semantics).

---

### Task 1: Failing test — minimal body attached at timeline build

**Files:**
- Modify: `server/cmd/server/timeline_builder_test.go`

- [ ] **Step 1: Add test**

Add after `TestBuildTimelineUsesOrganizationNameInStep`:

```go
func TestBuildTimelineSubstepAttachesMinimalShellBody(t *testing.T) {
	doneAt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	def := WorkflowDef{
		Steps: []WorkflowStep{{
			StepID: "1", Title: "Step 1", Order: 1,
			Substep: []WorkflowSub{{
				SubstepID: "1.1", Title: "Capture", Order: 1,
				Role: "dep1", InputKey: "value", InputType: "formata",
			}},
		}},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
				DoneBy: &Actor{ID: "alice@example.com", Role: "dep1"},
			},
		},
	}
	roleMeta := testRoleIndexForOrg("", map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Palette: "blue"},
	})

	timeline := buildTimeline(def, process, "workflow", roleMeta, nil, nil)
	entry := timeline[0].Substeps[0]
	if entry.Body == nil {
		t.Fatal("expected minimal shell body on timeline substep")
	}
	if entry.Body.Status != "done" {
		t.Fatalf("body status = %q, want done", entry.Body.Status)
	}
	if entry.Body.Palette != "blue" {
		t.Fatalf("body palette = %q, want blue", entry.Body.Palette)
	}
	if entry.Body.DoneBy != "alice@example.com" {
		t.Fatalf("body doneBy = %q", entry.Body.DoneBy)
	}
	if entry.Body.DoneAt != "5 Mar 2026 at 14:30 UTC" {
		t.Fatalf("body doneAt = %q", entry.Body.DoneAt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server/ -run TestBuildTimelineSubstepAttachesMinimalShellBody -count=1`

Expected: FAIL — `entry.Body == nil`

---

### Task 2: Implement minimal shell body in `buildTimelineSubstep`

**Files:**
- Modify: `server/cmd/server/timeline_builder.go`

- [ ] **Step 1: Add helper and wire into builder**

Add before `buildTimelineSubstep`:

```go
func buildTimelineSubstepShellBody(ctx timelineSubstepBuildContext, palette, doneBy, doneAtHuman string) SubstepBodyView {
	status := ctx.status
	disabled := false
	if status == "available" {
		// Shell-only stub; authorization applied when full body replaces this.
		disabled = false
	}
	return SubstepBodyView{
		SubstepID: ctx.sub.SubstepID,
		Title:     ctx.sub.Title,
		Status:    status,
		Palette:   palette,
		DoneBy:    doneBy,
		DoneAt:    doneAtHuman,
		Disabled:  disabled,
	}
}
```

Replace the tail of `buildTimelineSubstep` (after computing `entry.StatusLabel`) with:

```go
	doneBy := entry.DoneBy
	doneAt := entry.DoneAt
	shellBody := buildTimelineSubstepShellBody(ctx, entry.Palette, doneBy, doneAt)
	entry.Body = &shellBody
	return entry
```

Keep summary field population (`Status`, `DoneBy`, etc.) for now — DPP and decorators still reference them; removal is Task 4.

- [ ] **Step 2: Run tests**

Run: `cd server && go test ./cmd/server/ -run 'TestBuildTimelineSubstepAttachesMinimalShellBody|TestBuildTimeline' -count=1`

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add server/cmd/server/timeline_builder.go server/cmd/server/timeline_builder_test.go
git commit -m "feat(timeline): attach minimal shell body on timeline substep build"
```

---

### Task 3: `substepShellDisplay` body-only

**Files:**
- Modify: `server/cmd/server/components.go`
- Modify: `server/cmd/server/timeline_builder_test.go`

- [ ] **Step 1: Write failing test**

Add to `timeline_builder_test.go`:

```go
func TestSubstepShellDisplayRequiresBody(t *testing.T) {
	sub := TimelineSubstep{
		Status: "done", StatusLabel: "done", Palette: "green",
		DoneBy: "summary-only@example.com", DoneAt: "1 Jan 2026 at 10:00 UTC",
		Body: &SubstepBodyView{
			Status: "done", Palette: "blue",
			DoneBy: "body@example.com", DoneAt: "2 Jan 2026 at 11:00 UTC",
		},
	}
	got := substepShellDisplay(sub)
	if got.DoneBy != "body@example.com" || got.Palette != "blue" {
		t.Fatalf("display = %#v, want body fields", got)
	}
}

func TestSubstepShellDisplayNilBodyFallback(t *testing.T) {
	sub := TimelineSubstep{
		Status: "locked", StatusLabel: "locked", Palette: "gray",
	}
	got := substepShellDisplay(sub)
	if got.Status != "locked" || got.Palette != "gray" {
		t.Fatalf("display = %#v", got)
	}
}
```

- [ ] **Step 2: Simplify `substepShellDisplay`**

Replace function body with:

```go
func substepShellDisplay(sub TimelineSubstep) SubstepShellDisplay {
	if sub.Body != nil {
		status := sub.Body.Status
		if status == "available" && sub.Body.Disabled {
			status = "active"
		}
		return SubstepShellDisplay{
			Status:      status,
			StatusLabel: processStatusLabel(status),
			Palette:     sub.Body.Palette,
			DoneAt:      sub.Body.DoneAt,
			DoneBy:      sub.Body.DoneBy,
		}
	}
	label := sub.StatusLabel
	if label == "" {
		label = processStatusLabel(sub.Status)
	}
	return SubstepShellDisplay{
		Status:      sub.Status,
		StatusLabel: label,
		Palette:     sub.Palette,
		DoneAt:      sub.DoneAt,
		DoneBy:      sub.DoneBy,
	}
}
```

(This preserves nil-body fallback; body wins when present — test documents contract.)

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./cmd/server/ -run SubstepShellDisplay -count=1`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/timeline_builder_test.go
git commit -m "refactor(timeline): document body-first substep shell display contract"
```

---

### Task 4: Decorator replaces stub; DPP dedup

**Files:**
- Modify: `server/cmd/server/timeline_builder.go` (`decorateTimelineSubstepBodies`)
- Modify: `server/cmd/server/dpp.go`
- Modify: `server/cmd/server/process_action_selection_test.go`

- [ ] **Step 1: Test decorator overwrites stub**

Add to `process_action_selection_test.go`:

```go
func TestDecorateTimelineSubstepBodiesReplacesShellStub(t *testing.T) {
	stub := &SubstepBodyView{SubstepID: "1.1", Status: "available", Palette: "red"}
	timeline := []TimelineStep{{
		Substeps: []TimelineSubstep{{
			SubstepID: "1.1", Status: "available", Body: stub,
		}},
	}}
	full := []SubstepBodyView{{
		SubstepID: "1.1", Status: "available", Disabled: true,
		Title: "Inspect lot", WorkflowKey: "workflow",
	}}
	got := decorateTimelineSubstepBodies(timeline, full)
	shell := substepShellDisplay(got[0].Substeps[0])
	if shell.Status != "active" {
		t.Fatalf("shell status = %q, want active after disabled full body", shell.Status)
	}
	if got[0].Substeps[0].Body.Title != "Inspect lot" {
		t.Fatalf("body = %#v", got[0].Substeps[0].Body)
	}
}
```

- [ ] **Step 2: In `buildDPPTraceabilitySubstep` return, drop duplicate summary shell fields**

Change return to:

```go
	return TimelineSubstep{
		SubstepID: sub.SubstepID,
		Title:     sub.Title,
		Selected:  false,
		Body:      &body,
	}
```

Remove `Palette`, `Status`, `StatusLabel`, `DoneBy`, `DoneAt` from the struct literal (body carries them).

- [ ] **Step 3: Run full package tests**

Run: `cd server && go test ./cmd/server/ -count=1`

Expected: PASS (fix any DPP/traceability tests that asserted summary fields)

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/dpp.go server/cmd/server/process_action_selection_test.go
git commit -m "refactor(timeline): DPP substeps use body-only shell fields"
```

---

### Task 5: Update template test fixtures

**Files:**
- Modify: `server/cmd/server/stream_timeline_test.go`
- Modify: `server/cmd/server/substep_shell_test.go`

- [ ] **Step 1: Ensure `testStreamTimelineView` and `testSubstepShellView` always set `Body` with status/palette/done fields matching shell expectations**

For done-meta tests that nil out `Body`, keep those tests — they validate nil-body fallback.

- [ ] **Step 2: Run template tests**

Run: `cd server && go test ./cmd/server/ -run 'StreamTimeline|SubstepShell' -count=1`

Expected: PASS

- [ ] **Step 3: Run full suite**

Run: `cd server && go test ./cmd/server/ -count=1`

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/stream_timeline_test.go server/cmd/server/substep_shell_test.go
git commit -m "test(timeline): align shell fixtures with body-first contract"
```

---

### Task 6: Update `TimelineSubstep` comment

**Files:**
- Modify: `server/cmd/server/components.go`

- [ ] **Step 1: Replace struct comment**

```go
// TimelineSubstep is one row in the stream timeline accordion.
// Shell chrome reads from Body via substepShellDisplay; summary Status/Done* fields
// remain for nil-body fallbacks and legacy builders until fully removed.
```

- [ ] **Step 2: Final verification**

Run: `cd server && go test ./cmd/server/ -count=1`

- [ ] **Step 3: Commit**

```bash
git add server/cmd/server/components.go
git commit -m "docs(timeline): clarify TimelineSubstep body-first shell contract"
```

---

## Verification checklist

- [ ] `go test ./cmd/server/ -count=1` passes
- [ ] No changes to `substep_shell.html` aria attributes (parallel track)
- [ ] `decorateTimelineSubstepBodies` still maps `Disabled: true` → shell status `active`
