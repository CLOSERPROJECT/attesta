<script lang="ts">
	import { onMount } from 'svelte';

	import { takeColdEntityIdRaw } from '../org/qr-resolve-apply.js';
	import { createHttpQrResolvePort } from '../org/http-qr-resolve-port.js';
	import { orgSession } from '../org/org-app.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { runQrScan } from '../org/qr-scan-flow.js';
	import { orgToast } from '../org/toast.svelte.js';
	import { orgMyStreamDashboardPath, navigateOrgTarget } from './navigate-target.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);
	const qrPort = createHttpQrResolvePort();

	onMount(() => {
		const { raw, strippedHref } = takeColdEntityIdRaw(window.location.href);
		if (raw) {
			window.history.replaceState(window.history.state, '', strippedHref);
			void runQrScan(
				{
					resolve: qrPort.resolve,
					stream,
					drafts: orgSession,
					navigate: navigateOrgTarget,
					toast: (message) => orgToast.show(message)
				},
				raw
			);
			return;
		}
		if (stream) {
			window.location.replace(orgMyStreamDashboardPath(stream));
		} else {
			window.location.replace('/my');
		}
	});
</script>

<p class="text-sm text-muted-foreground" data-testid="org-stream-redirect">Redirecting…</p>
