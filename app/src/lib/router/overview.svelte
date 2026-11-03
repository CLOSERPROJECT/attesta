<script lang="ts">
	import { onMount } from 'svelte';

	import { living } from '../living/living-app.svelte.js';
	import WorklistRow from '../living/WorklistRow.svelte';
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

		<ul class="flex flex-col gap-1" data-testid="overview-tasks">
			{#each model.tasks as row (row.taskId)}
				<li data-testid="overview-task-row" data-task-id={row.taskId}>
					<WorklistRow
						data-testid={row.linkable ? 'overview-task-link' : 'overview-task-status'}
						ready={row.status === 'available'}
						href={row.linkable
							? p('/:jobId/:jobInstanceId/:taskId', {
									params: {
										jobId: model.jobId,
										jobInstanceId: model.jobInstanceId,
										taskId: row.taskId
									}
								})
							: null}
					>
						<p class="text-sm font-medium text-zinc-900">{row.title}</p>
						<p class="mt-0.5 font-mono text-xs text-zinc-500">
							{row.taskId} · {row.status}
						</p>
					</WorklistRow>
				</li>
			{/each}
		</ul>
	{/if}
</div>
