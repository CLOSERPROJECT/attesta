<script lang="ts">
	import { onMount } from 'svelte';

	import { living } from '../living/living-app.svelte.js';
	import {
		formatHandoffCopy,
		jobInstanceOverview
	} from '../living/worklist-overview.js';
	import { navigate, p, route } from './index.js';

	const jobId = $derived(String(route.params.jobId ?? ''));
	const jobInstanceId = $derived(String(route.params.jobInstanceId ?? ''));

	onMount(() => {
		if (!living.result && !living.loading) {
			void living.hydrate();
		}
	});

	const model = $derived(
		jobInstanceOverview(living.result?.worklist, jobId, jobInstanceId)
	);

	$effect(() => {
		const id = jobInstanceId;
		if (id === 'new' || id === '') {
			navigate('/');
			return;
		}
		if (living.loading || !living.result) return;
		if (!model || model.finished) {
			navigate('/');
		}
	});
</script>

<div class="mx-auto max-w-3xl space-y-4" data-testid="overview-route">
	<header class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<p class="font-mono text-xs text-zinc-500">{jobId} / {jobInstanceId}</p>
			<h1 class="text-xl font-semibold text-zinc-900" data-testid="overview-label">
				{model?.label ?? 'Job instance'}
			</h1>
		</div>
		<button
			type="button"
			class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-700 hover:bg-zinc-50"
			data-testid="overview-back-home"
			onclick={() => navigate('/')}
		>
			Home
		</button>
	</header>

	{#if living.loading && !living.result}
		<p class="text-sm text-zinc-500" data-testid="overview-load-status">Loading…</p>
	{:else if living.error && !living.result}
		<p class="text-sm text-red-700" data-testid="overview-load-error" role="alert">
			{living.error}
		</p>
	{:else if model}
		{#if model.handoffRoles}
			<p
				class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900"
				data-testid="overview-handoff"
			>
				{formatHandoffCopy(model.handoffRoles)}
			</p>
		{/if}

		<ul
			class="divide-y divide-zinc-200 rounded-lg border border-zinc-200 bg-white"
			data-testid="overview-tasks"
		>
			{#each model.tasks as row (row.taskId)}
				<li class="px-4 py-3" data-testid="overview-task-row" data-task-id={row.taskId}>
					{#if row.linkable}
						<a
							class="block hover:underline"
							data-testid="overview-task-link"
							href={p('/:jobId/:jobInstanceId/:taskId', {
								params: {
									jobId: model.jobId,
									jobInstanceId: model.jobInstanceId,
									taskId: row.taskId
								}
							})}
						>
							<p class="text-sm font-medium text-zinc-900">{row.title}</p>
							<p class="mt-0.5 font-mono text-xs text-zinc-500">
								{row.taskId} · {row.status}
							</p>
						</a>
					{:else}
						<div data-testid="overview-task-status">
							<p class="text-sm font-medium text-zinc-900">{row.title}</p>
							<p class="mt-0.5 font-mono text-xs text-zinc-500">
								{row.taskId} · {row.status}
							</p>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>
