<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import { p } from '../router/index.js';
	import { hrefLivingTarget } from '../router/navigate-target.js';
	import { draftAddressForTask } from './drafts.js';
	import { living } from './living-app.svelte.js';
	import WorklistJobLine from './WorklistJobLine.svelte';
	import WorklistRow from './WorklistRow.svelte';
	import {
		availableRowKey,
		availableRowsFromWorklist,
		availableRowTarget,
		canStartRowsFromWorklist,
		completedRowKey,
		completedRowsFromWorklist,
		inProgressRowKey,
		inProgressRowsFromWorklist,
		taskRouteInstanceSegment
	} from './worklist-home.js';

	onMount(() => {
		void living.hydrate();
	});

	const available = $derived(availableRowsFromWorklist(living.result?.worklist));
	const canStart = $derived(canStartRowsFromWorklist(living.result?.worklist));
	const inProgress = $derived(inProgressRowsFromWorklist(living.result?.worklist));
	const completed = $derived(completedRowsFromWorklist(living.result?.worklist));
	const drafts = $derived(living.drafts);

	function rowHasDirtyDraft(row: (typeof available)[number]): boolean {
		void drafts;
		const addr = draftAddressForTask({
			jobId: row.jobId,
			taskId: row.taskId,
			jobInstanceId: row.jobInstanceId,
			groupId: row.groupId,
			groupInstanceId: row.groupInstanceId
		});
		return living.hasDirtyDraft(addr.taskPath, addr.draftBagInstanceKey);
	}
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="living-shell">
	{#if !living.result && living.loading}
		<p class="text-sm text-zinc-500" data-testid="living-load-status">{m.loading()}</p>
	{:else if !living.result && living.error}
		<p class="text-sm text-red-700" data-testid="living-load-error" role="alert">
			{living.error}
		</p>
	{:else if !living.result}
		<p class="text-sm text-zinc-500" data-testid="living-load-status">
			{m.not_loaded_yet()}
		</p>
	{/if}
	{#if living.commitError}
		<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
			{m.commit_denied()}: {living.commitError}
		</p>
	{/if}
	{#if living.committing}
		<p class="text-sm text-zinc-500" data-testid="living-commit-status">{m.committing()}</p>
	{/if}

	{#if living.result}
		<section class="space-y-3" data-testid="worklist-available">
			<h2 class="text-lg font-semibold text-zinc-900">{m.available()}</h2>
			<ul class="flex flex-col gap-1">
				{#each available as row (availableRowKey(row))}
					<li>
						<WorklistRow
							data-testid="available-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							data-group-id={row.groupId}
							data-group-instance-id={row.groupInstanceId}
							data-task-id={row.taskId}
							ready
							showDraft={rowHasDirtyDraft(row)}
							accentColor={row.jobColor}
							href={hrefLivingTarget(availableRowTarget(row))}
						>
							<p class="min-w-0 text-sm font-bold text-zinc-900">{row.title}</p>
							<WorklistJobLine
								class="mt-1"
								mode="progress"
								size="xs"
								jobId={row.jobId}
								jobTitle={row.jobTitle}
								label={row.instanceLabel}
								done={row.tasksDone}
								total={row.tasksTotal}
								progressTestId="available-progress"
							/>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="available-empty" class="text-sm text-zinc-500">
							{m.no_available_tasks()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>

		<section class="space-y-3" data-testid="worklist-can-start">
			<h2 class="text-lg font-semibold text-zinc-900">{m.can_start()}</h2>
			<ul class="flex flex-col gap-1">
				{#each canStart as row (row.jobId)}
					<li>
						<WorklistRow
							data-testid="can-start-row"
							data-job-id={row.jobId}
							accentColor={row.jobColor}
							href={p('/:jobId/:jobInstanceId/:taskId', {
								params: {
									jobId: row.jobId,
									jobInstanceId: taskRouteInstanceSegment(''),
									taskId: row.firstTaskId
								}
							})}
						>
							<WorklistJobLine
								mode="start"
								size="sm"
								jobId={row.jobId}
								jobTitle={row.jobTitle}
								label={row.startLabel}
							/>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="can-start-empty" class="text-sm text-zinc-500">
							{m.no_jobs_to_start()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>

		<section class="space-y-3" data-testid="worklist-in-progress">
			<h2 class="text-lg font-semibold text-zinc-900">{m.in_progress()}</h2>
			<ul class="flex flex-col gap-1">
				{#each inProgress as row (inProgressRowKey(row))}
					<li>
						<WorklistRow
							data-testid="in-progress-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							accentColor={row.jobColor}
							href={hrefLivingTarget({
								kind: 'overview',
								jobId: row.jobId,
								jobInstanceId: row.jobInstanceId
							})}
						>
							<WorklistJobLine
								mode="progress"
								size="sm"
								jobId={row.jobId}
								jobTitle={row.jobTitle}
								label={row.instanceLabel}
								done={row.tasksDone}
								total={row.tasksTotal}
								progressTestId="in-progress-progress"
							/>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="in-progress-empty" class="text-sm text-zinc-500">
							{m.no_jobs_in_progress()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>

		<section class="space-y-3" data-testid="worklist-completed">
			<h2 class="text-lg font-semibold text-zinc-900">{m.completed()}</h2>
			<ul class="flex flex-col gap-1">
				{#each completed as row (completedRowKey(row))}
					<li>
						<WorklistRow
							data-testid="completed-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							accentColor={row.jobColor}
							href={hrefLivingTarget({
								kind: 'overview',
								jobId: row.jobId,
								jobInstanceId: row.jobInstanceId
							})}
						>
							<WorklistJobLine
								mode="progress"
								size="sm"
								jobId={row.jobId}
								jobTitle={row.jobTitle}
								label={row.instanceLabel}
								done={row.tasksDone}
								total={row.tasksTotal}
								progressTestId="completed-progress"
							/>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="completed-empty" class="text-sm text-zinc-500">
							{m.no_completed_jobs()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
