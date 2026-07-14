# Stream Timeline Accessibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve keyboard and screen-reader support for the nested step/substep accordion tree in `stream_timeline` without changing Go builder logic.

**Architecture:** Use native `<details>`/`<summary>` semantics plus explicit `aria-labelledby` / descriptive labels derived from stable fields (`StepID`, `Title`, `SubstepID`). Add minimal JS focus management when a substep panel opens on the process page. Coordinate with parallel body-first track: read labels from `.Substep.Title` / step summary title, not from status strings.

**Tech Stack:** Go `html/template`, `web/src/main.js`, CSS unchanged unless focus-visible gaps found.

**Worktree branch:** `feature/stream-timeline-a11y`

---

## File map

| File | Responsibility |
|------|----------------|
| `server/templates/components/stream_timeline.html` | Step-level accessible names |
| `server/templates/components/substep_shell.html` | Substep accordion ids + aria |
| `server/cmd/server/stream_timeline_test.go` | Assert aria markup |
| `server/cmd/server/substep_shell_test.go` | Assert aria markup |
| `web/src/main.js` | Focus substep summary after open (process page only) |

**Do not modify:** `timeline_builder.go`, `components.go` shell display logic (parallel track).

---

### Task 1: Step summary accessible name

**Files:**
- Modify: `server/templates/components/stream_timeline.html`
- Modify: `server/cmd/server/stream_timeline_test.go`

- [ ] **Step 1: Write failing template test**

Add to `stream_timeline_test.go`:

```go
func TestStreamTimelineStepSummaryHasAccessibleName(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", testStreamTimelineView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`id="stream-timeline-step-1"`,
		`aria-labelledby="stream-timeline-step-1-summary"`,
		`id="stream-timeline-step-1-summary"`,
		`Production`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in output, got: %s", want, body)
		}
	}
	if strings.Contains(body, `aria-label="Open stream step"`) {
		t.Fatal("generic aria-label should be replaced by aria-labelledby")
	}
}
```

- [ ] **Step 2: Update `stream_timeline_step` template**

Replace the `<details>` block opening:

```html
{{ define "stream_timeline_step" }}
  {{ $stepDomID := printf "stream-timeline-step-%s" .Step.Summary.StepID }}
  {{ $summaryDomID := printf "%s-summary" $stepDomID }}
  <details
    class="stream-timeline-step"
    id="{{ $stepDomID }}"
    {{ if .Step.Expanded }}open{{ end }}
  >
    <summary
      class="stream-timeline-step-summary"
      id="{{ $summaryDomID }}"
      aria-labelledby="{{ $summaryDomID }}"
    >
```

(Remove the generic `aria-label="Open stream step"`.)

- [ ] **Step 3: Run test**

Run: `cd server && go test ./cmd/server/ -run TestStreamTimelineStepSummaryHasAccessibleName -count=1`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/templates/components/stream_timeline.html server/cmd/server/stream_timeline_test.go
git commit -m "a11y(timeline): give stream steps stable ids and labelled summaries"
```

---

### Task 2: Substep accordion accessible name

**Files:**
- Modify: `server/templates/components/substep_shell.html`
- Modify: `server/cmd/server/substep_shell_test.go`

- [ ] **Step 1: Write failing test**

Add to `substep_shell_test.go`:

```go
func TestSubstepShellTemplateAccessibleAccordionName(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`id="substep-1-1-heading"`,
		`aria-labelledby="substep-1-1-heading"`,
		`Capture batch data`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
}
```

Note: substep IDs contain dots (`1.1`); DOM ids must replace `.` with `-` for valid HTML ids.

- [ ] **Step 2: Update `substep_shell.html`**

At top of define, add:

```html
{{ $substepDomID := replace .Substep.SubstepID "." "-" }}
```

Update accordion summary title:

```html
<span class="substep-title-heading" id="substep-{{ $substepDomID }}-heading">{{ .Substep.Title }}</span>
```

Update `<details>`:

```html
<details
  class="substep-accordion js-process-substep-panel"
  data-substep-id="{{ .Substep.SubstepID }}"
  aria-labelledby="substep-{{ $substepDomID }}-heading"
  {{ if .Substep.Selected }}open{{ end }}
>
```

**Important:** Go templates have no built-in `replace`. Register in `templateFuncs()` in `server/cmd/server/templates.go`:

```go
"replace": func(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
},
```

Add `"strings"` import if not present in `templates.go`.

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./cmd/server/ -run 'SubstepShell|StreamTimeline' -count=1`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/templates/components/substep_shell.html server/cmd/server/templates.go server/cmd/server/substep_shell_test.go
git commit -m "a11y(timeline): label substep accordions with stable heading ids"
```

---

### Task 3: Status pill accessibility

**Files:**
- Modify: `server/templates/components/substep_shell.html`
- Modify: `server/cmd/server/substep_shell_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestSubstepShellStatusHasAccessibleLabel(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `<span class="status" aria-label="Status: available">`) {
		t.Fatalf("expected status aria-label, got: %s", body)
	}
}
```

- [ ] **Step 2: Update status span in `substep_shell.html`**

```html
<span class="status" aria-label="Status: {{- if $shell.StatusLabel }}{{ $shell.StatusLabel }}{{ else }}{{ $shell.Status }}{{ end -}}">
  {{- if $shell.StatusLabel }}{{ $shell.StatusLabel }}{{ else }}{{ $shell.Status }}{{ end -}}
</span>
```

- [ ] **Step 3: Run tests + commit**

Run: `cd server && go test ./cmd/server/ -run SubstepShell -count=1`

```bash
git add server/templates/components/substep_shell.html server/cmd/server/substep_shell_test.go
git commit -m "a11y(timeline): expose substep status text to assistive tech"
```

---

### Task 4: Focus management on substep open

**Files:**
- Modify: `web/src/main.js`

- [ ] **Step 1: In the existing `toggle` listener for `.js-process-substep-panel`, after `markSelectedSubstep(substepID)`, focus the opened panel's summary**

Inside the `if (target.open)` block, after `markSelectedSubstep(substepID);`:

```javascript
    const summary = target.querySelector(".substep-accordion-summary");
    if (summary instanceof HTMLElement) {
      summary.focus();
    }
```

- [ ] **Step 2: Make substep summary focusable**

In `substep_shell.html`, add `tabindex="-1"` to `.substep-accordion-summary` so JS can focus it without adding to tab order:

```html
<summary class="substep-accordion-summary" tabindex="-1">
```

Add template test:

```go
func TestSubstepShellSummaryIsFocusable(t *testing.T) {
	// assert tabindex="-1" on substep-accordion-summary
}
```

- [ ] **Step 3: Run tests**

Run: `cd server && go test ./cmd/server/ -run SubstepShell -count=1`

- [ ] **Step 4: Commit**

```bash
git add web/src/main.js server/templates/components/substep_shell.html server/cmd/server/substep_shell_test.go
git commit -m "a11y(timeline): focus substep summary when panel opens"
```

---

### Task 5: Full verification

- [ ] **Step 1: Run full Go test suite**

Run: `cd server && go test ./cmd/server/ -count=1`

- [ ] **Step 2: Build frontend (sanity)**

Run: `cd web && npm run build`

Expected: success (JS change only)

- [ ] **Step 3: Manual smoke (optional)**

On process page: open substep accordion → summary receives focus; VoiceOver/NVDA reads step/substep titles.

---

## Verification checklist

- [ ] No Go builder files modified
- [ ] Template tests cover ids, aria-labelledby, status aria-label
- [ ] `replace` template func registered once in `templates.go`
- [ ] DOM ids sanitize `.` in substep IDs
