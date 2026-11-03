<script lang="ts">
	import { onMount } from 'svelte';

	import { p } from '../router/index.js';
	import { living } from './living-app.svelte.js';
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
	<header class="space-y-2">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-2xl font-semibold text-zinc-900">Home</h1>
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
				<span class="font-medium text-zinc-800">{living.result.configId}</span>
				v{living.result.activeConfigVersion}
			</p>
		{:else if living.loading}
			<p class="text-sm text-zinc-500" data-testid="living-load-status">Loading…</p>
		{:else if living.error}
			<p class="text-sm text-red-700" data-testid="living-load-error" role="alert">
				{living.error}
			</p>
		{:else}
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
	</header>

	{#if living.result}
		<section class="space-y-3" data-testid="worklist-available">
			<h2 class="text-lg font-semibold text-zinc-900">Available</h2>
			<ul class="divide-y divide-zinc-200 rounded-lg border border-zinc-200 bg-white">
				{#each available as row (availableRowKey(row))}
					{@const ringFrac = progressFraction(row.tasksDone, row.tasksTotal)}
					{@const ringOffset = PROGRESS_RING_C * (1 - ringFrac)}
					<li>
						<a
							class="block px-4 py-3 hover:bg-zinc-50"
							data-testid="available-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							data-task-id={row.taskId}
							href={p('/:jobId/:jobInstanceId/:taskId', {
								params: {
									jobId: row.jobId,
									jobInstanceId: row.jobInstanceId,
									taskId: row.taskId
								}
							})}
						>
							<div class="flex items-start justify-between gap-3">
								<p class="min-w-0 text-sm font-bold text-zinc-900">
									{row.instanceLabel}
									<span class="font-normal text-zinc-400">·</span>
									{#if row.jobTitle === row.jobId}
										<span class="font-mono text-xs font-normal text-zinc-500">{row.jobId}</span>
									{:else}
										<span class="text-xs font-normal text-zinc-500">{row.jobTitle}</span>
									{/if}
								</p>
								<span
									class="mt-1.5 size-2 shrink-0 rounded-full bg-emerald-500"
									role="img"
									aria-label="Available"
									data-testid="available-ready-dot"
								></span>
							</div>
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
						</a>
					</li>
				{:else}
					<li
						class="px-4 py-3 text-sm text-zinc-500"
						data-testid="available-empty"
					>
						No available Tasks.
					</li>
				{/each}
			</ul>
		</section>

		<section class="space-y-3" data-testid="worklist-can-start">
			<h2 class="text-lg font-semibold text-zinc-900">Can start</h2>
			<ul class="divide-y divide-zinc-200 rounded-lg border border-zinc-200 bg-white">
				{#each canStart as row (row.jobId)}
					<li>
						<a
							class="block px-4 py-3 hover:bg-zinc-50"
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
						</a>
					</li>
				{:else}
					<li
						class="px-4 py-3 text-sm text-zinc-500"
						data-testid="can-start-empty"
					>
						No Jobs to start.
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
