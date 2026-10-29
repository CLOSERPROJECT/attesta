<script lang="ts">
	import { onMount } from 'svelte';

	import JobChrome from './JobChrome.svelte';
	import { living } from './living-app.svelte.js';

	onMount(() => {
		void living.hydrate();
	});
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="living-shell">
	<header class="space-y-2">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-2xl font-semibold text-zinc-900">Jobs</h1>
			<button
				type="button"
				class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
				data-testid="living-reload"
				disabled={living.loading}
				onclick={() => void living.hydrate()}
			>
				{living.loading ? 'Loading…' : 'Reload'}
			</button>
		</div>
		{#if living.result}
			<p class="text-sm text-zinc-600" data-testid="living-load-meta">
				Org <span class="font-medium text-zinc-800">{living.result.org}</span>
				· Config
				<span class="font-medium text-zinc-800">{living.result.config.id}</span>
				v{living.result.activeConfigVersion}
			</p>
		{:else if living.loading}
			<p class="text-sm text-zinc-500" data-testid="living-load-status">Loading Domain…</p>
		{:else if living.error}
			<p class="text-sm text-red-700" data-testid="living-load-error" role="alert">
				{living.error}
			</p>
		{:else}
			<p class="text-sm text-zinc-500" data-testid="living-load-status">
				Domain not loaded yet.
			</p>
		{/if}
	</header>

	{#if living.result}
		<div class="space-y-4" data-testid="living-jobs">
			{#each living.jobIds as jobId (jobId)}
				{@const job = living.job(jobId)}
				{#if job}
					<JobChrome {jobId} {job} domain={living.result.domain} />
				{/if}
			{/each}
		</div>
	{/if}
</div>
