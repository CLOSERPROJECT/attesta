<script lang="ts">
	import IntakeJob from './IntakeJob.svelte';
	import LineJob from './LineJob.svelte';
	import { jobInstancesFor, type Domain, type JobConfig } from './types.js';

	type Props = {
		jobId: string;
		job: JobConfig;
		domain: Domain;
	};

	let { jobId, job, domain }: Props = $props();

	const taskIds = $derived(Object.keys(job.tasks ?? {}));
	const instances = $derived(jobInstancesFor(domain, jobId));
</script>

{#if jobId === 'intake'}
	<IntakeJob {jobId} {job} {domain} />
{:else if jobId === 'line'}
	<LineJob {jobId} {job} {domain} />
{:else}
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

		{#if instances.length > 0}
			<ul class="mt-3 space-y-1 text-sm text-zinc-600" data-testid="job-instances">
				{#each instances as inst (inst.id)}
					<li>
						<span class="font-medium text-zinc-800">{inst.label || inst.id}</span>
						<span class="text-zinc-400"> · {inst.id}</span>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="mt-3 text-sm text-zinc-500">No Job instances yet.</p>
		{/if}
	</section>
{/if}
