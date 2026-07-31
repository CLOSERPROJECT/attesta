# Reusable Success Toast Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a reusable, zero-dependency success toast (opt-in via `data-toast`) and wire it to all four categories-editor success confirmations without layout shift.

**Architecture:** Server keeps rendering a normal `.confirmation` node. Categories marks it with `data-toast`. CSS takes `[data-toast]` out of document flow from first paint (fixed toast placement). A small `main.js` helper promotes marked nodes into `#toast-host` on load and `htmx:afterSwap`, then auto-dismisses.

**Tech Stack:** Go `html/template`, HTMX, Vite CSS (`toast.css`), vanilla JS in `web/src/main.js` (no new npm deps).

**Spec:** `docs/superpowers/specs/2026-07-30-reusable-success-toast-design.md`

## Global Constraints

- Work only in git worktree `.worktrees/feat/admin-categories-crud` on branch `feat/admin-categories-crud`.
- No toast library / no new runtime dependency in `web/package.json`.
- Opt-in only via `data-toast`; unmarked `.confirmation` banners stay in-flow.
- No layout shift: `.confirmation[data-toast]` must be out of normal flow from first paint (CSS), before JS runs.
- Errors stay inline (`.error`) — never toasted.
- No Go toast DTO, no OOB HTMX fragment, no changes to confirmation query-param helpers.
- v1 consumer = categories panel only (create / edit / delete / reorder share one confirmation node).
- Read `docs/css.md` and `.agents/skills/attesta-ui-components` before CSS/template work (CSS-only tier for toast).
- English copy only; reuse existing handler confirmation strings.
- TDD where testable (Go template markup): failing test → implement → pass → commit per task.
- Prefer focused `go test` filters; full package may hit unrelated OpenAPI noise.

---

## File map

| File | Responsibility |
|------|----------------|
| `server/templates/layout.html` | Empty `#toast-host.toast-host` with `aria-live="polite"` |
| `web/src/styles/components/toast.css` | CSS-only toast host + out-of-flow `[data-toast]` / `.toast` styles |
| `web/src/styles/components.css` | Import `toast.css` |
| `web/src/main.js` | Promote `[data-toast]` into host; stack/cap/dismiss |
| `server/templates/pages/platform_admin.html` | Add `data-toast` on categories confirmation `<p>` |
| `server/cmd/server/layout_toast_test.go` | Assert layout includes toast host |
| `server/cmd/server/platform_admin_template_test.go` | Assert categories confirmation includes `data-toast` |

---

### Task 1: Toast host + CSS (no layout shift)

**Files:**
- Create: `web/src/styles/components/toast.css`
- Modify: `web/src/styles/components.css` (add import)
- Modify: `server/templates/layout.html` (insert host before `</body>`)
- Create: `server/cmd/server/layout_toast_test.go`

**Interfaces:**
- Consumes: existing tokens `--success-muted`, `--success-muted-foreground`, `--space-*`, `--foreground` / card tokens as needed
- Produces:
  - Markup host: `<div id="toast-host" class="toast-host" aria-live="polite"></div>`
  - CSS contract (file header): `.toast-host` + `.confirmation[data-toast]` / `.toast` out-of-flow fixed corner
  - Selector that takes marked confirmations out of flow **before JS**: `.confirmation[data-toast]`

- [ ] **Step 1: Write the failing layout host test**

Create `server/cmd/server/layout_toast_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutIncludesToastHost(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `id="toast-host"`) {
		t.Fatalf("expected #toast-host in layout, got:\n%s", body)
	}
	if !strings.Contains(body, `class="toast-host"`) {
		t.Fatalf("expected .toast-host class in layout, got:\n%s", body)
	}
	if !strings.Contains(body, `aria-live="polite"`) {
		t.Fatalf("expected aria-live=polite on toast host, got:\n%s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud/server && go test ./cmd/server/ -run 'TestLayoutIncludesToastHost' -count=1
```

Expected: FAIL — body missing `#toast-host`.

- [ ] **Step 3: Add host to layout**

In `server/templates/layout.html`, immediately before `</body>` (after `</footer>`), insert:

```html
      <div id="toast-host" class="toast-host" aria-live="polite"></div>
```

- [ ] **Step 4: Add toast CSS module + barrel import**

Create `web/src/styles/components/toast.css`:

```css
/*
 * Toast — fixed success notices (CSS-only component).
 *
 * div#toast-host.toast-host[aria-live=polite]
 *   p.confirmation.toast[data-toast]   (promoted by main.js)
 *
 * Opt-in source (still in page HTML until promoted; must not shift layout):
 *   p.confirmation[data-toast]
 *
 * Unmarked .confirmation banners stay in shared.css / document flow.
 */

.toast-host {
  position: fixed;
  z-index: 50;
  inset: auto var(--space-4) var(--space-4) auto;
  display: flex;
  flex-direction: column-reverse;
  gap: var(--space-2);
  width: min(22rem, calc(100vw - 2 * var(--space-4)));
  pointer-events: none;
}

.toast-host > * {
  pointer-events: auto;
}

/* Out of flow from first paint — no panel jump before JS promote */
.confirmation[data-toast] {
  position: fixed;
  z-index: 50;
  inset: auto var(--space-4) var(--space-4) auto;
  width: min(22rem, calc(100vw - 2 * var(--space-4)));
  margin: 0;
  box-shadow: 0 8px 24px color-mix(in srgb, var(--foreground) 12%, transparent);
  transition:
    opacity 160ms ease,
    transform 160ms ease;
}

.confirmation[data-toast].toast-enter,
.confirmation[data-toast].toast-exit {
  opacity: 0;
  transform: translateY(0.5rem);
}

.toast-host .confirmation[data-toast] {
  position: relative;
  inset: auto;
  width: 100%;
}
```

In `web/src/styles/components.css`, add (near other component imports, e.g. after `dialog.css`):

```css
@import url("./components/toast.css");
```

- [ ] **Step 5: Run layout test to verify it passes**

Run:

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud/server && go test ./cmd/server/ -run 'TestLayoutIncludesToastHost' -count=1
```

Expected: PASS

Also run CSS lint:

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud && task css:lint
```

Expected: PASS (or only pre-existing unrelated warnings — fix any new toast.css issues).

- [ ] **Step 6: Commit**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud
git add server/templates/layout.html server/cmd/server/layout_toast_test.go web/src/styles/components/toast.css web/src/styles/components.css
git commit -m "$(cat <<'EOF'
Add fixed toast host and CSS with no layout shift.

EOF
)"
```

---

### Task 2: Categories opt-in (`data-toast`)

**Files:**
- Modify: `server/templates/pages/platform_admin.html` (`platform_admin_categories_panel` confirmation block ~lines 165–167)
- Modify: `server/cmd/server/platform_admin_template_test.go` (add confirmation toast test)

**Interfaces:**
- Consumes: `CategoriesEditorView.Confirmation` (unchanged string field)
- Produces: markup `<p class="confirmation" data-toast>{{ .CategoriesEditor.Confirmation }}</p>` when confirmation is non-empty

- [ ] **Step 1: Write the failing template test**

Append to `server/cmd/server/platform_admin_template_test.go`:

```go
func TestPlatformAdminCategoriesConfirmationUsesDataToast(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			Confirmation: "Subcategory reordered",
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_panel", view); err != nil {
		t.Fatalf("render platform_admin_categories_panel: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Subcategory reordered") {
		t.Fatalf("expected confirmation text, got:\n%s", body)
	}
	if !strings.Contains(body, `class="confirmation" data-toast`) &&
		!strings.Contains(body, `data-toast`) {
		t.Fatalf("expected confirmation to opt into data-toast, got:\n%s", body)
	}
	// Prefer exact attribute adjacency used in template:
	if !strings.Contains(body, `<p class="confirmation" data-toast>`) {
		t.Fatalf("expected <p class=\"confirmation\" data-toast>, got:\n%s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud/server && go test ./cmd/server/ -run 'TestPlatformAdminCategoriesConfirmationUsesDataToast' -count=1
```

Expected: FAIL — confirmation present without `data-toast`.

- [ ] **Step 3: Opt in the categories confirmation**

In `server/templates/pages/platform_admin.html`, change:

```html
    {{ if .CategoriesEditor.Confirmation }}
      <p class="confirmation">{{ .CategoriesEditor.Confirmation }}</p>
    {{ end }}
```

to:

```html
    {{ if .CategoriesEditor.Confirmation }}
      <p class="confirmation" data-toast>{{ .CategoriesEditor.Confirmation }}</p>
    {{ end }}
```

Do **not** add `data-toast` to orgs / other confirmation banners in this task.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud/server && go test ./cmd/server/ -run 'TestPlatformAdminCategoriesConfirmationUsesDataToast|TestPlatformAdminCategories|TestAdminCategories' -count=1
```

Expected: PASS (handler tests still find confirmation text in the HTML response).

- [ ] **Step 5: Commit**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud
git add server/templates/pages/platform_admin.html server/cmd/server/platform_admin_template_test.go
git commit -m "$(cat <<'EOF'
Opt categories success confirmations into data-toast.

EOF
)"
```

---

### Task 3: Promote / dismiss in `main.js`

**Files:**
- Modify: `web/src/main.js`

**Interfaces:**
- Consumes: `#toast-host`, nodes matching `.confirmation[data-toast]` not already inside `#toast-host`
- Produces:
  - `function promoteToasts(root = document) { … }`
  - Called from existing `DOMContentLoaded` listener and existing `htmx:afterSwap` listener
  - Behavior: move into host (prepend so newest is on top with `flex-direction: column-reverse`), add enter class briefly, auto-dismiss after 3000ms with exit class, click dismisses early, cap at 3 toasts (remove oldest DOM node when over cap)

- [ ] **Step 1: Add toast helpers near other DOM helpers in `main.js`**

Place near the bottom of the file (before or after the existing `htmx:afterSwap` block is fine; keep functions hoisted or defined before use):

```js
const TOAST_DISMISS_MS = 3000;
const TOAST_MAX = 3;

function toastHost() {
  return document.getElementById("toast-host");
}

function dismissToast(el) {
  if (!(el instanceof HTMLElement) || el.dataset.toastDismissing === "1") {
    return;
  }
  el.dataset.toastDismissing = "1";
  el.classList.add("toast-exit");
  window.setTimeout(() => {
    el.remove();
  }, 180);
}

function enforceToastCap(host) {
  const items = [...host.querySelectorAll(".confirmation[data-toast]")];
  while (items.length > TOAST_MAX) {
    const oldest = items.shift();
    if (oldest) {
      oldest.remove();
    }
  }
}

function promoteToasts(root = document) {
  const host = toastHost();
  if (!(host instanceof HTMLElement)) {
    return;
  }
  const scope = root instanceof Element || root instanceof Document ? root : document;
  const nodes = scope.querySelectorAll?.(".confirmation[data-toast]") ?? [];
  for (const node of nodes) {
    if (!(node instanceof HTMLElement)) {
      continue;
    }
    if (host.contains(node)) {
      continue;
    }
    node.classList.add("toast-enter");
    host.prepend(node);
    enforceToastCap(host);
    requestAnimationFrame(() => {
      node.classList.remove("toast-enter");
    });
    node.addEventListener(
      "click",
      () => {
        dismissToast(node);
      },
      { once: true },
    );
    window.setTimeout(() => {
      dismissToast(node);
    }, TOAST_DISMISS_MS);
  }
}
```

- [ ] **Step 2: Wire promote into existing lifecycle hooks**

In the existing `DOMContentLoaded` listener (`document.addEventListener("DOMContentLoaded", () => { … })`), add:

```js
  promoteToasts(document);
```

In the existing `htmx:afterSwap` listener, after the current body (including the `#platform-admin-categories` block), add:

```js
  if (event.target instanceof Element) {
    promoteToasts(event.target);
  } else {
    promoteToasts(document);
  }
```

If `formatLocalDateTimes` already guards `event.target instanceof Element`, reuse that branch and call `promoteToasts(event.target)` inside it **and** also call `promoteToasts(event.target)` when the categories panel swaps (same target). One call per swap is enough:

```js
document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.target instanceof Element) {
    formatLocalDateTimes(event.target);
    promoteToasts(event.target);
  }
  // …existing process-page-content and platform-admin-categories blocks unchanged…
});
```

- [ ] **Step 3: Sanity-check (no JS unit harness in repo)**

There is no Vitest/Jest in `web/package.json`. Do not add a test runner for this task.

Manual check from worktree host-dev (`task start` then `task dev` if needed; URL from `.env.local`, e.g. `http://localhost:3153`):

1. Open `/admin/categories` as platform admin.
2. Reorder a subcategory ↑/↓ — toast appears bottom-right; **panel meta/list does not jump**.
3. Create a leaf — toast with create confirmation; auto-dismiss ~3s.
4. Trigger an error (e.g. blocked delete) — error stays **inline**, not a toast.
5. Rapid reorder several times — at most ~3 toasts stacked; oldest dropped.

- [ ] **Step 4: Run focused Go regression + CSS lint**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud/server && go test ./cmd/server/ -run 'TestLayoutIncludesToastHost|TestPlatformAdminCategoriesConfirmationUsesDataToast|TestAdminCategories' -count=1
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud && task css:lint
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.worktrees/feat/admin-categories-crud
git add web/src/main.js
git commit -m "$(cat <<'EOF'
Promote data-toast confirmations into the toast host.

EOF
)"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Opt-in promote; server still renders confirmation | 2, 3 |
| `data-toast` marker | 2 |
| No layout shift from first paint | 1 (CSS) |
| Errors stay inline | 2 (no error changes), 3 QA |
| No library / no new runtime dep | all |
| No Go API / OOB / query helper changes | all |
| `#toast-host` + `aria-live=polite` | 1 |
| Auto-dismiss ~3s, click dismiss, cap ~3 | 3 |
| HTMX `afterSwap` + initial load | 3 |
| Categories first consumer (all four via one node) | 2 |
| Template test for `data-toast` | 2 |
| Handler confirmation text still in response | 2 regression |
| Worktree `feat/admin-categories-crud` | Global Constraints |
