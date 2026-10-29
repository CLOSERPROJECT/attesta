# 02: Commit skeleton — register Task (test Authorizer)

**What to build:** Commit accepts a sealed leaf Task at Task path `intake/register` (plus Job instance id), persists Submission and Domain mutate via Memory DomainStore, and Load reflects the result. Authorizer is a test stub that allows; Cerbos is not required yet. Placeholder `POST /api/living/commit`.

**Blocked by:** 01: Living Load — empty Domain + fixture Org Config

**Status:** ready-for-agent

- [ ] Commit for `intake/register` updates Domain and records a Submission keyed by Task path
- [ ] Subsequent Load shows the committed Domain
- [ ] Request shape uses Task path + Job instance id (not a run id)
- [ ] Verified session required; tests cover the happy path with stub Authorizer
