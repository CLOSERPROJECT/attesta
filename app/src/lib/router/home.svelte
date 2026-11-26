<script lang="ts">
	import OrgBoard from '../org/board/org-board.svelte';
	import type { BoardScope } from '../org/board/board-session.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);
	const instance = $derived(coords.instance);
	const scope = $derived.by((): BoardScope | null =>
		stream && instance ? { kind: 'instance', stream, instance } : null
	);
</script>

{#if scope}
	<OrgBoard {scope} />
{:else}
	<p class="text-sm text-muted-foreground" data-testid="org-home-missing">Missing stream or instance.</p>
{/if}
