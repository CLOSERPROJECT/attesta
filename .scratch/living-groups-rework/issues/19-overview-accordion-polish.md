# 19: Overview accordion polish (experiment chrome)

**What to build:** Job instance overview visual/UX polish to match attesta-experiment chrome:

1. Task group accordion **expanded by default**
2. Hover: **primary border** (not background change)
3. Chevron **invisible but present** (visibility/opacity, not `hidden`/unmounted) when status is `locked`
4. Status badge styling/colors like attesta-experiment (`bloccato`/`disponibile`/`fatto` + Per te)
5. Close group confirm → **shadcn Dialog**, not inline confirm UI
6. Chevron **vertically centered**
7. Number box **smaller** (experiment pill size)
8. Overall row chrome closer to experiment `SubstepBlockChrome` / `step-chrome.css`
9. **No separator** between group instances list and Spawn/Close buttons

**Blocked by:** None (builds on 17–18)

**Status:** ready-for-agent

- [ ] Group default open
- [ ] Hover primary border
- [ ] Locked chevron invisible-but-laid-out
- [ ] Experiment-colored status badges
- [ ] Close confirm Dialog
- [ ] Chevron vertical align + smaller number box
- [ ] No separator before Spawn/Close
