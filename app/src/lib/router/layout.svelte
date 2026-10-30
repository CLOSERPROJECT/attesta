<script lang="ts">
	import type { Snippet } from 'svelte';

	import { isActive, p } from './index.js';

	let { children }: { children: Snippet } = $props();
</script>

<nav class="flex gap-4 border-b border-zinc-200 px-4 py-3 text-sm">
	<a class:font-semibold={isActive('/')} href={p('/')}>Jobs</a>
	<a class:font-semibold={isActive('/about')} href={p('/about')}>About</a>
</nav>

<main class="p-4">
	{@render children()}
</main>
