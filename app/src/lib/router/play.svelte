<script lang="ts">
	import OrgBoard from '../org/board/org-board.svelte';
	import type { BoardScope } from '../org/board/board-session.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);
	const scope = $derived.by((): BoardScope | null =>
		stream ? { kind: 'boot', stream } : null
	);
</script>

{#if scope}
	<OrgBoard {scope} />
{:else}
	<p class="text-sm text-zinc-500" data-testid="play-missing">Missing stream.</p>
{/if}
