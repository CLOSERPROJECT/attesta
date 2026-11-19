<script lang="ts">
	import { onMount } from 'svelte';

	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';
	import { orgMyInstanceTimelinePath } from './navigate-target.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));

	onMount(() => {
		if (coords.stream && coords.instance) {
			window.location.replace(orgMyInstanceTimelinePath(coords.stream, coords.instance));
		} else {
			window.location.replace('/my');
		}
	});
</script>

<p class="text-sm text-zinc-500" data-testid="org-home-redirect">Redirecting…</p>
