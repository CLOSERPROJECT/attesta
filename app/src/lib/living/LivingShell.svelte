<script lang="ts">
	import { onMount } from 'svelte';

	import { p } from '../router/index.js';
	import { living } from './living-app.svelte.js';
	import WorklistRow from './WorklistRow.svelte';
	import {
		availableRowKey,
		availableRowsFromWorklist,
		canStartRowsFromWorklist,
		progressFraction,
		taskRouteInstanceSegment
	} from './worklist-home.js';

	onMount(() => {
		void living.hydrate();
	});

	const available = $derived(availableRowsFromWorklist(living.result?.worklist));
	const canStart = $derived(canStartRowsFromWorklist(living.result?.worklist));

	const PROGRESS_RING_R = 5.5;
	const PROGRESS_RING_C = 2 * Math.PI * PROGRESS_RING_R;
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="living-shell">
	{#if !living.result && living.loading}
		<p class="text-sm text-zinc-500" data-testid="living-load-status">Loading…</p>
	{:else if !living.result && living.error}
		<p class="text-sm text-red-700" data-testid="living-load-error" role="alert">
			{living.error}
		</p>
	{:else if !living.result}
		<p class="text-sm text-zinc-500" data-testid="living-load-status">
			Not loaded yet.
		</p>
	{/if}
	{#if living.commitError}
		<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
			Commit denied: {living.commitError}
		</p>
	{/if}
	{#if living.committing}
		<p class="text-sm text-zinc-500" data-testid="living-commit-status">Committing…</p>
	{/if}

	{#if living.result}
		<section class="space-y-3" data-testid="worklist-available">
			<h2 class="text-lg font-semibold text-zinc-900">Available</h2>
			<ul class="flex flex-col gap-1">
				{#each available as row (availableRowKey(row))}
					{@const ringFrac = progressFraction(row.tasksDone, row.tasksTotal)}
					{@const ringOffset = PROGRESS_RING_C * (1 - ringFrac)}
					<li>
						<WorklistRow
							data-testid="available-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							data-task-id={row.taskId}
							ready
							href={p('/:jobId/:jobInstanceId/:taskId', {
								params: {
									jobId: row.jobId,
									jobInstanceId: row.jobInstanceId,
									taskId: row.taskId
								}
							})}
						>
							<p class="min-w-0 text-sm font-bold text-zinc-900">
								{row.instanceLabel}
								<span class="font-normal text-zinc-400">·</span>
								{#if row.jobTitle === row.jobId}
									<span class="font-mono text-xs font-normal text-zinc-500">{row.jobId}</span>
								{:else}
									<span class="text-xs font-normal text-zinc-500">{row.jobTitle}</span>
								{/if}
							</p>
							<div class="mt-1 flex min-w-0 items-center gap-1.5">
								<p class="min-w-0 truncate text-sm text-zinc-600">{row.title}</p>
								<span
									class="inline-flex shrink-0 items-center gap-1"
									role="img"
									aria-label="{row.tasksDone} of {row.tasksTotal} tasks done"
									data-testid="available-progress"
									data-done={row.tasksDone}
									data-total={row.tasksTotal}
								>
									<svg
										class="size-3.5 -rotate-90 text-zinc-200"
										viewBox="0 0 14 14"
										aria-hidden="true"
									>
										<circle
											cx="7"
											cy="7"
											r={PROGRESS_RING_R}
											fill="none"
											stroke="currentColor"
											stroke-width="1.75"
										/>
										<circle
											class="text-zinc-700"
											cx="7"
											cy="7"
											r={PROGRESS_RING_R}
											fill="none"
											stroke="currentColor"
											stroke-width="1.75"
											stroke-linecap="round"
											stroke-dasharray={PROGRESS_RING_C}
											stroke-dashoffset={ringOffset}
										/>
									</svg>
									<span class="tabular-nums text-[11px] leading-none text-zinc-500">
										{row.tasksDone}/{row.tasksTotal}
									</span>
								</span>
							</div>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="available-empty" class="text-sm text-zinc-500">
							No available Tasks.
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>

		<section class="space-y-3" data-testid="worklist-can-start">
			<h2 class="text-lg font-semibold text-zinc-900">Can start</h2>
			<ul class="flex flex-col gap-1">
				{#each canStart as row (row.jobId)}
					<li>
						<WorklistRow
							data-testid="can-start-row"
							data-job-id={row.jobId}
							href={p('/:jobId/:jobInstanceId/:taskId', {
								params: {
									jobId: row.jobId,
									jobInstanceId: taskRouteInstanceSegment(''),
									taskId: row.firstTaskId
								}
							})}
						>
							<p class="text-sm font-medium text-zinc-900">{row.startLabel}</p>
							{#if row.jobTitle === row.jobId}
								<p class="mt-0.5 font-mono text-xs text-zinc-500">{row.jobId}</p>
							{:else}
								<p class="mt-0.5 text-xs text-zinc-500">{row.jobTitle}</p>
							{/if}
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="can-start-empty" class="text-sm text-zinc-500">
							No Jobs to start.
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
