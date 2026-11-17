<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import { p } from '../router/index.js';
	import { hrefOrgTarget } from '../router/navigate-target.js';
	import { draftAddressForTask } from './drafts.js';
	import { orgSession } from './org-app.svelte.js';
	import ManySubmitHistory from './many-submit-history.svelte';
	import WorklistJobLine from './worklist-job-line.svelte';
	import WorklistRow from './worklist-row.svelte';
	import {
		actNowRowKey,
		actNowRowsFromWorklist,
		availableRowTarget,
		taskRouteInstanceSegment,
		type ActNowRow
	} from './worklist-home.js';

	onMount(() => {
		void orgSession.hydrate();
	});

	const actNow = $derived(actNowRowsFromWorklist(orgSession.result?.worklist));
	const drafts = $derived(orgSession.drafts);

	function rowHasDirtyDraft(row: Extract<ActNowRow, { kind: 'task' }>): boolean {
		void drafts;
		const addr = draftAddressForTask({
			jobId: row.jobId,
			taskId: row.taskId,
			jobInstanceId: row.jobInstanceId
		});
		return orgSession.hasDirtyDraft(addr.taskPath, addr.draftBagInstanceKey);
	}
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="org-shell">
	{#if !orgSession.result && orgSession.loading}
		<p class="text-sm text-zinc-500" data-testid="org-load-status">{m.loading()}</p>
	{:else if !orgSession.result && orgSession.error}
		<p class="text-sm text-red-700" data-testid="org-load-error" role="alert">
			{orgSession.error}
		</p>
	{:else if !orgSession.result}
		<p class="text-sm text-zinc-500" data-testid="org-load-status">
			{m.not_loaded_yet()}
		</p>
	{/if}
	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
		</p>
	{/if}
	{#if orgSession.committing}
		<p class="text-sm text-zinc-500" data-testid="org-commit-status">{m.committing()}</p>
	{/if}

	{#if orgSession.result}
		<section class="space-y-3" data-testid="worklist-act-now">
			<ul class="flex flex-col gap-1">
				{#each actNow as row (actNowRowKey(row))}
					<li>
						{#if row.kind === 'task'}
							<div class="flex flex-col gap-1">
								<WorklistRow
									data-testid="available-row"
									data-job-id={row.jobId}
									data-instance-id={row.jobInstanceId}
									data-task-id={row.taskId}
									ready
									showDraft={rowHasDirtyDraft(row)}
									accentColor={row.jobColor}
									href={hrefOrgTarget(availableRowTarget(row))}
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
								<ManySubmitHistory
									submit={row.submit}
									submissionCount={row.submissionCount}
									request={{
										jobId: row.jobId,
										jobInstanceId: row.jobInstanceId,
										taskId: row.taskId
									}}
									data-testid="home-many-history"
								/>
							</div>
						{:else}
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
						{/if}
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="act-now-empty" class="text-sm text-zinc-500">
							{m.no_available_tasks()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
