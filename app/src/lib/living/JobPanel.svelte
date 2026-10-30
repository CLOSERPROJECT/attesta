<script lang="ts">
	import { living } from './living-app.svelte.js';
	import TaskForm from './TaskForm.svelte';
	import { findTaskFormBundle } from './form/task-form-bundle.js';
	import type { TaskStatus, WorklistJob } from './types.js';

	type Props = {
		jobId: string;
		worklistJob: WorklistJob;
	};

	let { jobId, worklistJob }: Props = $props();

	const taskIds = $derived(worklistJob.taskIds ?? []);
	const instances = $derived(worklistJob.instances ?? []);
	const startLabel = $derived(worklistJob.startLabel ?? 'Nuovo');
	const canStart = $derived(worklistJob.canStart ?? false);
	const bundles = $derived(living.result?.taskFormBundles ?? []);
	const provisionalBundle = $derived(findTaskFormBundle(bundles, jobId, taskIds[0] ?? '', ''));

	let provisional = $state(false);

	function status(taskId: string, instanceId: string): TaskStatus {
		const inst = instances.find((row) => row.id === instanceId);
		return inst?.tasks[taskId] ?? 'locked';
	}

	function openStart() {
		if (!canStart || !provisionalBundle) return;
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
			{#if worklistJob.startLabel}
				<p class="mt-0.5 text-sm text-zinc-500">{worklistJob.startLabel}</p>
			{/if}
		</div>
		{#if worklistJob.repeatable}
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

	{#if !canStart}
		<p class="mt-3 text-sm text-amber-800" data-testid="job-starts-when-blocked">
			Job start is blocked until its startsWhen gate is satisfied.
		</p>
	{/if}

	<div class="mt-4 flex justify-end">
		<button
			type="button"
			class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
			data-testid={`${jobId}-start`}
			disabled={living.committing || !canStart || !provisionalBundle}
			onclick={openStart}
		>
			{startLabel}
		</button>
	</div>

	{#if provisional && provisionalBundle}
		<div class="mt-4">
			<TaskForm
				jobId={provisionalBundle.jobId}
				taskId={provisionalBundle.taskId}
				jobInstanceId=""
				bundle={provisionalBundle}
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
						{@const taskStatus = status(tid, inst.id)}
						{@const taskBundle = findTaskFormBundle(bundles, jobId, tid, inst.id)}
						{#if taskStatus === 'available' && taskBundle}
							<div class="mt-3 border-t border-zinc-100 pt-3">
								<TaskForm
									jobId={taskBundle.jobId}
									taskId={taskBundle.taskId}
									jobInstanceId={inst.id}
									bundle={taskBundle}
								/>
							</div>
						{:else if taskStatus === 'done'}
							<p class="mt-2 text-xs text-zinc-500" data-testid={`task-${tid}-done`}>
								{tid} · done
							</p>
						{/if}
					{/each}
				</li>
			{/each}
		</ul>
	{/if}
</section>
