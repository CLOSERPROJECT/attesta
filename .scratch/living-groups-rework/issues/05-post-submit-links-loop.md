# 05: After submit with downloads — stay, Start another, Close

**What to build:** When Commit returns download links (e.g. lot labels), the operator stays on a read-only view of the submitted form with those links. For an open Task group, that screen also offers **Start another** (Spawn) and **Close** (confirm). After Close, navigation matches ticket 04 / current next law. Child Commits with no links still go to overview while the group is open. The stay-for-links behavior applies to any Task Commit that returns downloads, not only group children.

**Blocked by:** 03 — Job page — Spawn opens the first form; 04 — Job page — Close group (confirm) and advance

**Status:** done

- [x] Commit with download links does not navigate away; form is read-only and shows links
- [x] Open group: Start another starts the next provisional first-leaf form; Close uses confirm then next-Task / overview / home
- [x] Commit without links while group open still lands on overview
- [x] Flat (non-group) Commits with download links also stay read-only + links until the operator continues (continue uses existing next destination)
- [x] Tests cover with-links and without-links paths; Sidecar/Fake test doubles supply links in tests as needed
