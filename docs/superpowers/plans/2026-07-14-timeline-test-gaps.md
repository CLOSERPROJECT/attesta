# Timeline Test Gaps Implementation Plan (Architecture #5)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Close high-priority test gaps for stream timeline selection and status rendering.

**Architecture:** Table-driven unit tests for `decorateTimelineSelection`; handler test asserting expanded/selected markup from `/content?substep=`; template tests for terminated/skipped shell classes.

**Tech Stack:** Go `testing`, `httptest`, existing `parseTestTemplates` / `testRuntimeConfig` / `MemoryStore`.

**Branch:** `refactor/domain-vocabulary` (main workspace — no worktree)

---

### Task 1: `decorateTimelineSelection` unit tests

**Files:**
- Modify: `server/cmd/server/timeline_builder_test.go`

- [ ] **Step 1: Add test**

```go
func TestDecorateTimelineSelection(t *testing.T) {
	timeline := []TimelineStep{
		{
			Summary: StepSummaryView{StepID: "1"},
			Substeps: []TimelineSubstep{
				{SubstepID: "1.1"},
				{SubstepID: "1.2"},
			},
		},
		{
			Summary: StepSummaryView{StepID: "2"},
			Substeps: []TimelineSubstep{
				{SubstepID: "2.1"},
			},
		},
	}

	t.Run("selects substep and expands its step", func(t *testing.T) {
		got := decorateTimelineSelection(append([]TimelineStep(nil), timeline...), "1.2")
		if !got[0].Expanded {
			t.Fatal("step 1 should be expanded")
		}
		if got[0].Substeps[0].Selected || !got[0].Substeps[1].Selected {
			t.Fatalf("selection = %#v", got[0].Substeps)
		}
		if got[1].Expanded {
			t.Fatal("step 2 should not expand")
		}
	})

	t.Run("empty selection clears flags", func(t *testing.T) {
		got := decorateTimelineSelection(append([]TimelineStep(nil), timeline...), "")
		for _, step := range got {
			if step.Expanded {
				t.Fatalf("step %s expanded", step.Summary.StepID)
			}
			for _, sub := range step.Substeps {
				if sub.Selected {
					t.Fatalf("substep %s selected", sub.SubstepID)
				}
			}
		}
	})

	t.Run("unknown substep leaves timeline unchanged", func(t *testing.T) {
		got := decorateTimelineSelection(append([]TimelineStep(nil), timeline...), "9.9")
		for _, step := range got {
			if step.Expanded {
				t.Fatal("no step should expand for unknown substep")
			}
		}
	})
}
```

- [ ] **Step 2: Run**

`cd server && go test ./cmd/server/ -run TestDecorateTimelineSelection -count=1`

- [ ] **Step 3: Commit**

`git commit -m "test(timeline): cover decorateTimelineSelection"`

---

### Task 2: Handler test for content partial selection

**Files:**
- Modify: `server/cmd/server/process_read_handler_test.go` (or new `stream_timeline_handler_test.go`)

- [ ] **Step 1: Add test using existing server test harness**

Assert GET `/w/workflow/process/:id/content?substep=1.2` response contains:

- `data-selected-substep="1.2"` on `#process-page` (or equivalent in partial)
- `data-substep-id="1.2"` with `open` on accordion
- Step containing 1.2 expanded (`stream-timeline-step` with `open`)

Use `testRuntimeConfig()`, `MemoryStore`, seeded process with substeps 1.1 pending, 1.2 available.

- [ ] **Step 2: Run**

`cd server && go test ./cmd/server/ -run 'ProcessContent|ProcessRead' -count=1`

- [ ] **Step 3: Commit**

`git commit -m "test(timeline): assert substep selection in content partial"`

---

### Task 3: Template tests for terminated/skipped shell classes

**Files:**
- Modify: `server/cmd/server/substep_shell_test.go`

- [ ] **Step 1: Add tests**

Render `substep_shell` with `Body.Status` of `processStatusTerminated` and `"skipped"`; assert:

- `class="substep substep-<status>"` (or bare `substep` when HideStatus)
- Status pill text / aria-label includes human label

- [ ] **Step 2: Run**

`cd server && go test ./cmd/server/ -run SubstepShell -count=1`

- [ ] **Step 3: Commit**

`git commit -m "test(timeline): cover terminated and skipped substep shell markup"`

---

### Task 4: Verification

- [ ] `cd server && go test ./cmd/server/ -count=1`
- [ ] Update `docs/stream-timeline-improvements-backlog.md` — mark architecture #5 done

---

## Verification checklist

- [ ] All new tests pass
- [ ] No production code changes unless tests reveal a bug (fix minimally if so)
