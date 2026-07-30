# Public home shared topbar + footer legal mix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/` use the shared app topbar and keep the public-home footer shell with site legal copy in place of the link columns (nav columns parked in a component for later).

**Architecture:** Flip the `layout.html` topbar guard so `public_home_body` gets the same `.topbar` as other pages; delete the marketing `.public-home-header`. Keep skipping layout `.site-footer` on `/`. Move Platform/Developer/Enterprise markup into an unused `public_home_footer_nav` define; put the long AGPL/CLOSER legal paragraphs in the former columns slot.

**Tech Stack:** Go `html/template`, handler tests with `parseTestTemplates`, Vite CSS (`public-home.css`), `docs/css.md`.

**Spec:** `docs/superpowers/specs/2026-07-30-public-home-shared-topbar-design.md`

## Global Constraints

- Replace marketing header entirely with shared topbar (no Product/Solutions/Pricing/Docs in chrome).
- Signed-out topbar: Login only (no Get Started in topbar); in-page signup CTAs stay.
- Keep `page-landing` on `<main>`; keep skipping `.site-footer` for `public_home_body`.
- Park footer link columns in a component; do not `{{ template "public_home_footer_nav" }}` from the live page.
- Legal copy in the home footer must match layout `.site-footer` paragraphs (AGPL, CLOSER, Even Closer, EU disclaimer).
- Do not DRY legal copy into a shared partial in this plan (spec non-goal).

## File map

| File | Responsibility |
|------|----------------|
| `server/templates/layout.html` | Render `.topbar` for public home; still skip `.site-footer` |
| `server/templates/pages/public_home.html` | Remove header; footer brand + legal slot + bottom meta |
| `server/templates/components/public_home_footer_nav.html` | Parked link-column markup (`define "public_home_footer_nav"`) |
| `web/src/styles/pages/public-home.css` | Drop header rules; style `.public-home-footer-legal`; keep parked-cols rules for later |
| `docs/css.md` | Fix public-home chrome exception wording |
| `server/cmd/server/home_handler_test.go` | Expect shared topbar + legal footer; no marketing header/cols |

---

### Task 1: Restore shared topbar (remove marketing header)

**Files:**
- Modify: `server/cmd/server/home_handler_test.go` (`TestHandlePublicHomeIsBlankAndPublic`, `TestHandlePublicHomeShowsDashboardWhenLoggedIn`)
- Modify: `server/templates/layout.html` (topbar guard only)
- Modify: `server/templates/pages/public_home.html` (delete `<header class="public-home-header">…</header>`)
- Modify: `web/src/styles/pages/public-home.css` (header-related rules + file header comment)
- Modify: `docs/css.md` (public-home exception line)

**Interfaces:**
- Consumes: existing `PageBase` fields (`Body`, `ShowLogout`, `UserEmail`, …) already passed into `public_home.html` via layout
- Produces: `/` HTML includes `class="topbar"`; no `class="public-home-header"`

- [ ] **Step 1: Update failing anonymous public-home chrome assertions**

In `server/cmd/server/home_handler_test.go`, replace the chrome assertions inside `TestHandlePublicHomeIsBlankAndPublic` (keep status/redirect/`public-home`/hero checks) with:

```go
	body := rec.Body.String()
	if !strings.Contains(body, `class="topbar`) {
		t.Fatalf("expected shared topbar on public home, got %q", body)
	}
	if !strings.Contains(body, `id="theme-toggle"`) {
		t.Fatalf("expected theme toggle on public home, got %q", body)
	}
	if !strings.Contains(body, `href="/login"`) {
		t.Fatalf("expected Login link on public home, got %q", body)
	}
	if !strings.Contains(body, `>Login</a>`) {
		t.Fatalf("expected shared topbar Login label on public home, got %q", body)
	}
	if strings.Contains(body, `class="public-home-header"`) {
		t.Fatalf("expected no marketing header on public home, got %q", body)
	}
	if strings.Contains(body, `class="public-home-signin`) {
		t.Fatalf("expected no landing Sign In chrome on public home, got %q", body)
	}
	if strings.Contains(body, `class="site-footer"`) {
		t.Fatalf("expected no shared site-footer on public home, got %q", body)
	}
	if !strings.Contains(body, `class="public-home"`) {
		t.Fatalf("expected public landing markup, got %q", body)
	}
	if !strings.Contains(body, "Verified traceability for supply chains") {
		t.Fatalf("expected landing hero copy, got %q", body)
	}
	if strings.Contains(body, `account-menu`) {
		t.Fatalf("expected no account menu on public home, got %q", body)
	}
	if strings.Contains(body, `>Dashboard</a>`) {
		t.Fatalf("expected no Dashboard link when logged out, got %q", body)
	}
```

In `TestHandlePublicHomeShowsDashboardWhenLoggedIn`, replace the Sign In–specific negative checks with:

```go
	if strings.Contains(body, `class="public-home-header"`) {
		t.Fatalf("expected no marketing header when logged in, got %q", body)
	}
	if strings.Contains(body, `class="public-home-signin`) {
		t.Fatalf("expected no landing Sign In chrome when logged in, got %q", body)
	}
	if !strings.Contains(body, `class="topbar`) {
		t.Fatalf("expected shared topbar when logged in, got %q", body)
	}
	if strings.Contains(body, `>Login</a>`) {
		t.Fatalf("expected no Login link when logged in, got %q", body)
	}
```

Keep the existing `account-menu` / Dashboard positive assertions.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd server && go test ./cmd/server -run 'TestHandlePublicHomeIsBlankAndPublic|TestHandlePublicHomeShowsDashboardWhenLoggedIn' -count=1
```

Expected: FAIL — missing `class="topbar` / still has `public-home-header` or `public-home-signin`.

- [ ] **Step 3: Enable shared topbar in layout**

In `server/templates/layout.html`, change the topbar guard from:

```html
{{ if ne .Body "public_home_body" }}
  <header class="topbar ...
{{ end }}
```

to always render the topbar (delete the `if ne .Body "public_home_body"` wrapper around the `<header class="topbar">…</header>` block only). Leave the `.site-footer` guard as:

```html
{{ if ne .Body "public_home_body" }}
  <footer class="site-footer">
```

- [ ] **Step 4: Remove marketing header from public home**

In `server/templates/pages/public_home.html`, delete the entire block:

```html
    <header class="public-home-header">
      ...
    </header>
```

Keep `$asset` / `$brand` vars (still used by footer logo and social icons). Body should start:

```html
  <div class="public-home" data-landing>
    <section class="public-home-hero">
```

- [ ] **Step 5: Delete marketing-header CSS + update docs**

In `web/src/styles/pages/public-home.css`:

1. Update the file-header contract to remove `header.public-home-header`.
2. Change the shared padding selector from:

```css
.public-home-header,
.public-home-hero,
...
```

to omit `.public-home-header` (same for `@media (--md-up)` / `@media (--lg-up)` padding lists).
3. Delete rules for: `.public-home-header`, `.public-home-brand`, `.public-home-nav`, `.public-home-nav a:hover`, `.public-home-header-cta`, `.public-home-signin`, `.public-home-signin::-webkit-details-marker`, `.public-home-account`, `.public-home-account .account-dropdown`, and any `.public-home-header .btn-primary` / `.public-home-nav { display: flex }` media rules.
4. Keep `.public-home-logo` (footer brand image still uses it).

In `docs/css.md`, change the exception line from:

```markdown
- `pages/public-home.css` ↔ `pages/public_home.html` (marketing landing; own chrome, hides app topbar/footer)
```

to:

```markdown
- `pages/public-home.css` ↔ `pages/public_home.html` (marketing landing; uses shared app topbar, own footer instead of `.site-footer`)
```

- [ ] **Step 6: Run tests to verify they pass**

Run:

```bash
cd server && go test ./cmd/server -run 'TestHandlePublicHomeIsBlankAndPublic|TestHandlePublicHomeShowsDashboardWhenLoggedIn' -count=1
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add server/cmd/server/home_handler_test.go server/templates/layout.html server/templates/pages/public_home.html web/src/styles/pages/public-home.css docs/css.md
git commit -m "$(cat <<'EOF'
feat(home): restore shared topbar on public homepage

EOF
)"
```

---

### Task 2: Footer legal mix + park link columns

**Files:**
- Create: `server/templates/components/public_home_footer_nav.html`
- Modify: `server/templates/pages/public_home.html` (footer top columns → legal)
- Modify: `web/src/styles/pages/public-home.css` (`.public-home-footer-legal`)
- Modify: `server/cmd/server/home_handler_test.go` (footer assertions + parked define lookup)
- Test: same `home_handler_test.go`

**Interfaces:**
- Consumes: layout `.site-footer` paragraph copy (verbatim)
- Produces: `define "public_home_footer_nav"` (parsed, not invoked); live footer uses `.public-home-footer-legal`

- [ ] **Step 1: Write failing footer / parked-nav assertions**

Append to `TestHandlePublicHomeIsBlankAndPublic` (after existing body checks):

```go
	if !strings.Contains(body, `class="public-home-footer-legal"`) {
		t.Fatalf("expected legal prose slot in public home footer, got %q", body)
	}
	if !strings.Contains(body, "GNU AGPLv3") {
		t.Fatalf("expected AGPL legal copy in public home footer, got %q", body)
	}
	if !strings.Contains(body, "Project No. 101161109") {
		t.Fatalf("expected CLOSER funding copy in public home footer, got %q", body)
	}
	if !strings.Contains(body, "Project No. 101228240") {
		t.Fatalf("expected Even Closer funding copy in public home footer, got %q", body)
	}
	if strings.Contains(body, `class="public-home-footer-heading"`) {
		t.Fatalf("expected parked footer nav columns not rendered, got %q", body)
	}
	if strings.Contains(body, `>Platform</p>`) {
		t.Fatalf("expected Platform footer heading not rendered, got %q", body)
	}
	if server.tmpl.Lookup("public_home_footer_nav") == nil {
		t.Fatal("expected parked template define public_home_footer_nav")
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd server && go test ./cmd/server -run 'TestHandlePublicHomeIsBlankAndPublic$' -count=1
```

Expected: FAIL — missing `public-home-footer-legal` / AGPL / parked define.

- [ ] **Step 3: Create parked footer nav component**

Create `server/templates/components/public_home_footer_nav.html`:

```html
{{ define "public_home_footer_nav" }}
  <div class="public-home-footer-cols">
    <div>
      <p class="public-home-footer-heading">Platform</p>
      <a href="#streams">Streams</a>
      <a href="#features">Pricing</a>
      <a href="#how-it-works">Security</a>
    </div>
    <div>
      <p class="public-home-footer-heading">Developer</p>
      <a href="#developers">OpenAPI Specs</a>
      <a href="https://github.com/forkbomb/attesta" rel="noopener noreferrer">GitHub</a>
      <a href="#developers">API Docs</a>
    </div>
    <div>
      <p class="public-home-footer-heading">Enterprise</p>
      <a href="https://closer-project.eu/" rel="noopener noreferrer">EU CLOSER</a>
      <a href="#developers">EIC Grants</a>
      <a href="/signup">Contact</a>
    </div>
  </div>
{{ end }}
```

Do **not** add `{{ template "public_home_footer_nav" }}` anywhere in `public_home.html`.

- [ ] **Step 4: Swap live footer columns for legal prose**

In `server/templates/pages/public_home.html`, replace the `<div class="public-home-footer-cols">…</div>` block inside `.public-home-footer-top` with:

```html
        <div class="public-home-footer-legal">
          <p>
            © 2025-2026 Forkbomb bv (forkbomb.eu) — The Forkbomb Company. Licensed
            under the GNU AGPLv3.
          </p>
          <p>
            CLOSER (Circular raw materiaLs for european Open Strategic autonomy on
            chips and microElectronics pRoduction, Project No. 101161109) is
            funded by the European Union under the Interregional Innovation
            Investments (I3) Instrument of the European Regional Development Fund,
            managed by the European Innovation Council and SMEs Executive Agency
            (EISMEA).
          </p>
          <p>
            Even Closer (Project No. 101228240) is funded by the European Union
            under the Interregional Innovation Investments (I3) Instrument of the
            European Regional Development Fund, managed by the European Innovation
            Council and SMEs Executive Agency (EISMEA).
          </p>
          <p>
            This repository/website is part of CLOSER and Even Closer projects and has
            received funding from the European Union. Views and opinions expressed are
            those of the author(s) only and do not necessarily reflect those of
            the European Union or EISMEA. Neither the European Union nor the
            granting authority can be held responsible for them.
          </p>
        </div>
```

Keep `.public-home-footer-brand` and `.public-home-footer-bottom` unchanged.

- [ ] **Step 5: Style the legal slot**

In `web/src/styles/pages/public-home.css`, add after `.public-home-footer-brand > p` (keep existing `.public-home-footer-cols*` rules for the parked component):

```css
.public-home-footer-legal {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  flex: 1;
  min-width: 0;
  color: var(--muted-foreground);
  font-size: var(--text-xs);
  line-height: var(--leading-normal);
}

.public-home-footer-legal p {
  margin: 0;
}
```

In `@media (--lg-up)` (or `--md-up` if that is where `.public-home-footer-top` becomes a row), ensure the brand + legal sit side-by-side using the existing `.public-home-footer-top` flex rules; if the top row is still `flex-direction: column` on large screens, set:

```css
@media (--md-up) {
  .public-home-footer-top {
    flex-direction: row;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-10);
  }
}
```

only if that rule is not already present — match current footer layout intent (brand left, content right). Do not invent a second `.site-footer` background/opacity treatment.

- [ ] **Step 6: Run tests to verify they pass**

Run:

```bash
cd server && go test ./cmd/server -run 'TestHandlePublicHomeIsBlankAndPublic|TestHandlePublicHomeShowsDashboardWhenLoggedIn' -count=1
```

Expected: PASS

Also confirm templates still parse:

```bash
cd server && go test ./cmd/server -run 'TestParseTemplates|TestHandlePublicStreamsPartial' -count=1
```

Expected: PASS (partial test must still assert no `public-home-header`).

- [ ] **Step 7: Commit**

```bash
git add server/templates/components/public_home_footer_nav.html server/templates/pages/public_home.html web/src/styles/pages/public-home.css server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
feat(home): put site legal copy in public home footer

EOF
)"
```

---

## Spec coverage (self-review)

| Spec decision | Task |
|---------------|------|
| Shared topbar for public home | Task 1 |
| Remove marketing header + CSS | Task 1 |
| Login only / signed-in account menu | Task 1 tests |
| Keep `page-landing`; skip `.site-footer` | Task 1 (layout footer guard unchanged; test asserts no site-footer) |
| Park footer nav component, not invoked | Task 2 |
| Legal copy in columns slot | Task 2 |
| Update `docs/css.md` | Task 1 |
| Handler test updates | Tasks 1–2 |

No placeholders left. Parked define name `public_home_footer_nav` is consistent across create/assert steps.
