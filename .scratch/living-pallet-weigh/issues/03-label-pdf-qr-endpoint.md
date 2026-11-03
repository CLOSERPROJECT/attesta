# 03: Label PDF embeds cold QR via $qrEndpoint

**What to build:** After intake lots register, the labels download PDF shows a QR per pallet whose payload is the Config-authored string built with `$qrEndpoint` (e.g. origin/app path + `?entityId=` + pallet id). Sidecar only encodes the `qr` field; Living injects `$qrEndpoint` as a submit/HttpPort binding from ops config. Cold phone-camera open of that https link is the goal of the payload shape.

**Blocked by:** None (can start immediately; parallel with 01–02).

**Status:** ready-for-agent

- [ ] Register HttpPort body can set per-label `qr` using `$qrEndpoint`
- [ ] `$qrEndpoint` is available in submit bindings from server/ops configuration
- [ ] Labels PDF renders a QR from each label’s `qr` string (dumb encode)
- [ ] Artifact link behaviour for “Stampa etichette” still works
