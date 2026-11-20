<script lang="ts">
	import OrgBoard from '../org/board/org-board.svelte';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);
</script>

{#if stream}
	<OrgBoard {stream} play={true} />
{:else}
	<p class="text-sm text-zinc-500" data-testid="play-missing">Missing stream.</p>
{/if}
