# Org Config declares post-Commit HTTP side effects as paths; platform owns peers and browser proxies

Post-Commit side effects need a way for Config to say *what* to fire without embedding peer hosts, secrets, or browser-reachable Adapter URLs.

**Decision:** **HttpPort** (`onSubmit.http`) runs after a successful Domain mutate. Config declares opaque **path-shaped** effect identifiers (template-evaluated); the platform joins each path to an Adapter via deployment env. Org Config never holds credentials or peer hostnames. Failures are soft: surfaced on the Commit response, Domain write is not rolled back. Go resolves path → Adapter, maps responses into Commit links, and rewrites artifact locations to Attesta URLs; the SPA/browser never call the Adapter—artifact GETs use an Attesta authorized proxy. Absolute `http(s)` values from evaluated Config are rejected (SSRF). The default peer Adapter is reversible; the locked seam is path-only Config, platform-owned resolution, proxy-only browser access, and soft failures.

**Considered options:** Absolute URLs in Config; browser fetches Adapter URLs; roll back Domain on HttpPort failure; secrets in Config; lock one plant sidecar forever; `data:` bodies as Commit links. Rejected for SSRF/secret sprawl, session bypass, mutate authority, and keeping the Adapter swappable.
