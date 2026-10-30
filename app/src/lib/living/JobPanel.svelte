<script lang="ts">
	import { living } from './living-app.svelte.js';
	import TaskForm from './TaskForm.svelte';
	import { canStartJob } from './starts-when.js';
	import { lineLoadLeftoverIds } from './leftovers.js';
	import {
		jobInstancesFor,
		type Domain,
		type JobConfig
	} from './types.js';
	import { isTaskUnlocked, orderedTaskIds, taskSubmitted } from './unlock.js';
	import {
		isLeftoverTableTask,
		isMapTableTask,
		needsLeftoverProjection
	} from './form/leftover-table-task.js';

	type Props = {
		jobId: string;
		job: JobConfig;
		domain: Domain;
	};

	let { jobId, job, domain }: Props = $props();

	const taskIds = $derived(orderedTaskIds(job.tasks));
	const instances = $derived(jobInstancesFor(domain, jobId));
	const startLabel = $derived(job.startLabel ?? 'Nuovo');
	const firstTaskId = $derived(taskIds[0] ?? '');
	const config = $derived(living.result?.config ?? null);

	const gateOpen = $derived(
		job.startsWhen ? canStartJob({ job, jobId, domain, config }) : true
	);

	let provisional = $state(false);

	function unlocked(taskId: string, instanceId: string): boolean {
		return isTaskUnlocked({
			orderedIds: taskIds,
			jobId,
			taskId,
			jobInstanceId: instanceId,
			submissions: domain.submissions,
			tasks: job.tasks ?? {}
		});
	}

	function done(taskId: string, instanceId: string): boolean {
		return taskSubmitted(domain, jobId, taskId, instanceId);
	}

	/** Show Task form: unlocked and (not done, or leftover-some still has rows). */
	function taskVisible(taskId: string, instanceId: string): boolean {
		if (!unlocked(taskId, instanceId)) return false;
		const authored = job.tasks?.[taskId];
		if (isLeftoverTableTask(authored)) {
			// lineLoad: keep open while shared leftovers remain
			if (jobId === 'line' && taskId === 'lineLoad') {
				return lineLoadLeftoverIds(domain).length > 0;
			}
			return !done(taskId, instanceId);
		}
		if (isMapTableTask(authored) || needsLeftoverProjection(authored)) {
			return !done(taskId, instanceId);
		}
		return !done(taskId, instanceId);
	}

	function openStart() {
		if (!gateOpen || !firstTaskId) return;
		provisional = true;
	}

	function onProvisionalCommitted(_id: string) {
		provisional = false;
	}
</script>

<section
	class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm"
	data-testid="job-chrome"
	data-job-id={jobId}
>
	<header class="flex flex-wrap items-baseline justify-between gap-2">
		<div>
			<h2 class="text-lg font-semibold text-zinc-900">{jobId}</h2>
			{#if job.startLabel}
				<p class="mt-0.5 text-sm text-zinc-500">{job.startLabel}</p>
			{/if}
		</div>
		{#if job.repeatable}
			<span class="text-xs font-medium tracking-wide text-zinc-500 uppercase">
				repeatable
			</span>
		{/if}
	</header>

	<ul class="mt-3 flex flex-wrap gap-2" data-testid="job-task-ids">
		{#each taskIds as tid (tid)}
			<li class="rounded-md bg-zinc-100 px-2 py-0.5 font-mono text-xs text-zinc-700">
				{jobId}/{tid}
			</li>
		{/each}
	</ul>

	{#if job.startsWhen && !gateOpen}
		<p class="mt-3 text-sm text-amber-800" data-testid="line-starts-when-blocked">
			Line starts after intake palletWeight (startsWhen some).
		</p>
	{/if}

	<div class="mt-4 flex justify-end">
		<button
			type="button"
			class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
			data-testid={`${jobId}-start`}
			disabled={living.committing || !gateOpen || !firstTaskId}
			onclick={openStart}
		>
			{startLabel}
		</button>
	</div>

	{#if provisional && firstTaskId}
		<div class="mt-4">
			<TaskForm
				{jobId}
				taskId={firstTaskId}
				jobInstanceId=""
				authored={job.tasks[firstTaskId]}
				{domain}
				onCommitted={onProvisionalCommitted}
			/>
			<button
				type="button"
				class="mt-2 rounded-md border border-zinc-300 px-3 py-1.5 text-sm"
				onclick={() => (provisional = false)}
			>
				Annulla
			</button>
		</div>
	{/if}

	{#if instances.length === 0}
		<p class="mt-3 text-sm text-zinc-500" data-testid="job-instances-empty">
			No Job instances yet.
		</p>
	{:else}
		<ul class="mt-4 space-y-4" data-testid="job-instances">
			{#each instances as inst (inst.id)}
				<li
					class="rounded-md border border-zinc-200 p-3"
					data-testid={`${jobId}-instance`}
					data-instance-id={inst.id}
				>
					<p class="text-sm font-medium text-zinc-900">
						{inst.label || inst.id}
						<span class="font-normal text-zinc-400"> · {inst.id}</span>
					</p>

					{#each taskIds as tid (tid)}
						{#if taskVisible(tid, inst.id)}
							<div class="mt-3 border-t border-zinc-100 pt-3">
								<TaskForm
									{jobId}
									taskId={tid}
									jobInstanceId={inst.id}
									authored={job.tasks[tid]}
									{domain}
								/>
							</div>
						{:else if done(tid, inst.id)}
							<p class="mt-2 text-xs text-zinc-500" data-testid={`task-${tid}-done`}>
								{tid} · submitted
							</p>
						{:else if tid === 'lineLoad' && unlocked(tid, inst.id) && lineLoadLeftoverIds(domain).length === 0}
							<p class="mt-2 text-xs text-zinc-500" data-testid="task-lineLoad-clear">
								lineLoad · leftovers clear
							</p>
						{/if}
					{/each}
				</li>
			{/each}
		</ul>
	{/if}
</section>
