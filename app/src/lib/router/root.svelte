<script lang="ts">
	import { onMount } from 'svelte';

	onMount(() => {
		window.location.replace('/my');
	});
</script>

<p class="text-sm text-zinc-500" data-testid="org-root-redirect">Redirecting…</p>
