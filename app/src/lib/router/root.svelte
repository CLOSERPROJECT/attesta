<script lang="ts">
	import { onMount } from 'svelte';

	onMount(() => {
		window.location.replace('/my');
	});
</script>

<p class="text-sm text-muted-foreground" data-testid="org-root-redirect">Redirecting…</p>
