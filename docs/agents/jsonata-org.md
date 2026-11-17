# jsonata-go gotchas (Org Config expr)

Org Config runs chrome / transform / QR / HttpPort bodies through [`jsonata-go`](https://github.com/blues/jsonata-go). Same surface as JSONata in spirit; these footguns are engine-specific. Prefer fixing exprs in Config over special-casing Go/SPA.

## Quirks

- **No `/* */` inside expr** — the engine treats that as regex, not a comment.
- **`$max` / `$min` take one array** — write `$max([$a, $b])`, not two args.
- **`$map` on empty → `null`** — not `[]`. Gate first, e.g. `$count($ids) = 0 ? [] : $map($ids, ...)`.
- **Empty maps** — same pattern with `$count($keys(...)) = 0` before mapping.
- **One-element `$each` unwraps to a bare object** — if a consumer expects an array (e.g. sidecar label entries), wrap: `[$each(...)]`.
- **`$exists` after `$lookup`** — can be true for missing keys; sift or check the field you care about.
- **Safe sums** — null/empty arrays need an explicit `$reduce` (or equivalent) helper; do not assume `$sum` alone is enough.

When an expr “works in a playground” but fails in Org, check this list before changing executor code.
