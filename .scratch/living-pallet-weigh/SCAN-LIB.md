# Living Scan — camera/decode library

**Decision (2026-11-03):** `cam2qr` owns camera + QR decode in Living ScanDialog.
Homemade `getUserMedia` + `BarcodeDetector` is retired (Safari/iOS gap).

## Constraints (pit of success)
- `ScanDialog.svelte` = dialog shell + paste fallback only; no raw `getUserMedia` / `BarcodeDetector`.
- Prefer `cam2qr/svelte` (`createQrScanner` + `use:video`); gate `<video>` with `{#if open}` so mount starts and unmount releases.
- Keep paste path + `POST { raw }` resolve unchanged (LOCKED QR resolve).
- Headless API only — do not adopt stock scanner UI.
- HTTPS / Tailscale Serve still required for camera (secure context).

## Pointers
- Component: `app/src/lib/living/ScanDialog.svelte`
- Resolve: `app/src/lib/living/qr-resolve-apply.ts`, `http-qr-resolve-port.ts`
- Phone smoke: `https://<MagicDNS>/login` → Scan
- Docs: https://cam2qr.com/ — `npm i cam2qr`
