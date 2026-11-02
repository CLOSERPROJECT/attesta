# 02: Living seed: split roles + staged Domain

**What to build:** `attesta seed` (living path) leaves Feragame with multiple operators and **Job instances** at different Submission stages so each login’s viewer-scoped Worklist looks different. Keep `living@attesta.com` as all-roles debug; add three split-role users (administration, logistic, line). Domain stages include at least: intake after `register`, after `codiceFir`, mid/`palletWeight` available, ready for `weightCheck`, and a `line` instance after `shift` (`lineLoad` available). Stream seed behavior stays as-is.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Seed creates (or updates) three Feragame users with single Config roles administration / logistic / line, plus existing all-roles `living@attesta.com`
- [x] Seeded Domain includes the minimal intake/line stage matrix above (Submissions + Job instances, not empty Domain only)
- [x] After seed, loading as each split-role user yields a distinct Available / Can start picture (API or UI)
- [x] `--replace` still wipes/rebuilds predictably; docs (QUICKSTART or equivalent) mention the living multi-actor seed briefly
- [x] Stream/process seed mix is unchanged
