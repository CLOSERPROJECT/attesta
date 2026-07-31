# Public stream card: Figma layout + metadata

Date: 2026-07-30  
Scope: `server/templates/components/public_stream_card.html`, `web/src/styles/components/public-stream-card.css`, `PublicStreamCardView` / `buildPublicStreamCardView`, Lucide icon template for DPP, tests in `public_stream_card_test.go` + `home_handler_test.go`  
Out of scope: Category sidebar / HTMX results, making the card a link, downloading assets from Figma, platform admin taxonomy, `/my` stream picker cards

Figma reference (structure/spacing; light tokens; metadata copy overridden below): [Stream card](https://www.figma.com/design/UYbx6DfxoFp9CCjqaD3qEr/CLOSER---Prototypes---Forkbomb?node-id=503-11899)

## Goal

Restyle the public homepage stream card to match the Figma shell (compact header + icon metrics + orgs), with product-specific metadata rows and a DPP chip under the description. Remove the Stream / Product Passport badges and the expanded Steps list.

## Decisions

1. **Approach:** Restyle existing `public_stream_card` in place (no v2 variant, no page-only CSS hacks).
2. **Badges:** Remove **Stream** and **Product Passport**. When DPP/passport is enabled, show a **DPP** chip under the description only.
3. **Zero runs:** First metrics row is a single line: `[icon] no runs yet`. Hide the active/completed metric.
4. **Roles count:** Distinct trimmed role slugs required across all workflow substeps (not the YAML role catalog length).
5. **Organizations count:** Show total org count as a small bold number next to the “Organizations” label — not a muted chip. Keep avatar row + `+N` overflow as today.
6. **Icons:** Existing Lucide templates in `icons.html`; add one small file/document icon for the DPP chip if missing. Do **not** download Figma MCP assets.
7. **Theme:** Light shared tokens (`--card`, `--border`, `--muted`, `--primary`, etc.). Do not adopt Figma’s dark palette.

## Layout

```
Title
Description
[DPP chip]                         ← only if PassportEnabled
[icon] N runs | [icon] N active now
        OR [icon] all completed    ← when T≥1 and no active
        OR [icon] no runs yet      ← when T=0 (alone)
[icon] N steps | [icon] N roles
─────────────
Organizations N                    ← N = total orgs; bold number, not a chip
avatars (+overflow)
```

Not a link. Card remains a static `<article>`.

### Metric copy

| Condition | First row |
|-----------|-----------|
| `InstanceCount == 0` | `[layers] no runs yet` |
| `AllCompleted` | `[layers] N run(s)` + `[check] all completed` |
| else | `[layers] N run(s)` + `[activity] N active now` |

Second row always: `[list] N steps` + `[users] N roles` (singular/plural as appropriate).

“Runs” is the existing instance count (`InstanceCount`); rename display copy from “instances” → “runs”.

### Icons

| Metric | Template |
|--------|----------|
| runs / no runs yet | `icon-layers-2` |
| active now | `icon-activity` (keep `--primary` accent class) |
| all completed | `icon-check-circle` |
| steps | `icon-list` |
| roles | `icon-users-group` |
| DPP chip | new Lucide-style `icon-file-text` (or equivalent) in `icons.html` |

## Visual / CSS

- Keep shell: flex column, `justify-content: space-between`, ~24px padding, 12px radius, `--card` / `--border`, `min-height: 360px` (already close to Figma).
- DPP chip: small muted badge under description (`--muted` surface, `--border`, ~12px text + file icon).
- Two metric rows with horizontal gaps (~12–16px); icons 14–16px; body text ~14px.
- Org head: “Organizations” + inline bold smaller number; no chip styling.
- Delete unused rules for the old badges row and steps list block; update the CSS file header contract comment.

## View model

`PublicStreamCardView`:

| Field | Role |
|-------|------|
| `Name`, `Description` | unchanged |
| `PassportEnabled` | drives DPP chip |
| `InstanceCount`, `ActiveCount`, `AllCompleted` | first metrics row |
| `StepCount` | `len(sortedSteps(workflow))` |
| `RoleCount` | distinct substep role slugs |
| `OrganizationCount` | total orgs before avatar truncation |
| `Organizations`, `OrganizationsOverflow` | unchanged avatar row |

**Remove:** `Steps` and `PublicStreamCardStepView` (card-only).

Builder (`buildPublicStreamCardView`) fills the new counts; instance/active/all-completed logic stays as today.

## Files

| Area | Change |
|------|--------|
| `public_stream_card.html` | New structure; drop badges + steps list |
| `public-stream-card.css` | Match layout; remove dead steps/badge rules |
| `components.go` | View fields; drop step row type |
| `public_home.go` | Builder counts |
| `icons.html` | DPP file icon if missing |
| `public_stream_card_test.go`, `home_handler_test.go` | Assert new markup; drop Stream/Passport/steps-list expectations |

## Testing

- Card renders title + description; no `Stream` / `Product Passport` badge classes.
- DPP chip present iff `PassportEnabled`; absent otherwise.
- Zero instances → `no runs yet`; no active/completed sibling.
- Settled (`AllCompleted`) → runs + `all completed`.
- Active → runs + `N active now`.
- Steps/roles metrics use `StepCount` / `RoleCount` (distinct roles across substeps).
- Organizations head includes bold total; avatars + overflow unchanged.
- No Figma asset URLs in templates or CSS.

## Success criteria

- Public home cards match Figma structure (compact metrics + orgs) on light tokens.
- Metadata rows match the product copy in Layout (runs / active|completed / steps / roles).
- DPP chip sits under description when enabled.
- Org total is a plain bold number, not a chip.
- Existing public-home sidebar/HTMX behavior unchanged.
