<script lang="ts">
	import OrgBoard from '../org/board/org-board.svelte';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);
	const instance = $derived(coords.instance);
</script>

{#if stream && instance}
	<OrgBoard {stream} {instance} />
{:else}
	<p class="text-sm text-zinc-500" data-testid="org-home-missing">Missing stream or instance.</p>
{/if}
