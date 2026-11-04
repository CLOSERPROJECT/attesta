# living-http

Plant HttpPort peer for Attesta Compose. Speaks the fixture paths
`/fake/labels` and `/fake/register-csv`; stores artifacts on disk.

## Response body shape (POST)

HttpPort treats the response body as a plain-text URL. Successful POSTs
return a path (no host):

```text
/artifacts/<unguessable-id>
```

Attesta rewrites that into `/api/living/artifacts/<id>` for the SPA.
Do not return `data:` URLs from this service.

GET `/artifacts/<id>` echoes `X-Attesta-Org` from stored meta when set so the
Attesta proxy can enforce session org match.

## Auth

All routes require `Authorization: Bearer <LIVING_HTTP_TOKEN>`.

## Org binding

Optional `X-Attesta-Org` header is stored with the artifact so Attesta can
enforce org match on proxy GET. Plant smoke may omit it until 01B sends it.

## Env

| Variable | Default | Purpose |
| --- | --- | --- |
| `LIVING_HTTP_TOKEN` | (required) | Shared secret with Attesta |
| `LIVING_HTTP_DATA_DIR` | `/data` | Artifact disk root (created on start) |
| `LIVING_HTTP_ADDR` | `:8090` | Listen address |

## Compose

Internal network only (`http://living-http:8090`). No host port by default.
Attesta gets `LIVING_HTTP_URL` + `LIVING_HTTP_TOKEN`.
After changing this package (e.g. `artifacts.go`), rebuild the image: `task living-http:rebuild`.
