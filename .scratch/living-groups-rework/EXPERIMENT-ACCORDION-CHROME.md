# Experiment accordion chrome (scout 2026-11-03)

Donor: `attesta-experiment` `step-chrome.css` + `SubstepBlockChrome.svelte`.

**Primary:** `#1e6f5c`. Hover: `border-color: primary` (no bg swap).

**Substep number pill:** mono `text-xs`, tight pad, `--pill: card`, `--border: foreground` (not 24×24 step pill).

**Status `.status`:** uppercase `text-xs` `font-semibold` `tracking-[0.04em]`; available → primary + 5% mix; done/locked → `#b3b3b3` + 5% mix.

**Card:** `radius 4px`, `border-left: 6px`; available muted `#e9f1ef` / `#d4dbd9`; done/locked `#f7f7f7` / `#d4d4d4`; locked whole-block `opacity: 0.3`.

**Chevron:** 32×32 hit, 18 icon; experiment omits on pending — living ticket 19 keeps layout, `invisible`/`opacity-0` when locked.
