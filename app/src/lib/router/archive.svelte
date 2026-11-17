<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import { orgSession } from '../org/org-app.svelte.js';
	import WorklistJobLine from '../org/worklist-job-line.svelte';
	import WorklistRow from '../org/worklist-row.svelte';
	import {
		completedRowKey,
		completedRowsFromWorklist,
		completedRowTarget
	} from '../org/worklist-home.js';
	import { hrefOrgTarget } from './navigate-target.js';

	onMount(() => {
		if (!orgSession.result && !orgSession.loading) {
			void orgSession.hydrate();
		}
	});

	const completed = $derived(completedRowsFromWorklist(orgSession.result?.worklist));
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="archive-route">
	{#if !orgSession.result && orgSession.loading}
		<p class="text-sm text-zinc-500" data-testid="archive-load-status">{m.loading()}</p>
	{:else if !orgSession.result && orgSession.error}
		<p class="text-sm text-red-700" data-testid="archive-load-error" role="alert">
			{orgSession.error}
		</p>
	{:else if !orgSession.result}
		<p class="text-sm text-zinc-500" data-testid="archive-load-status">
			{m.not_loaded_yet()}
		</p>
	{/if}

	{#if orgSession.result}
		<section class="space-y-3" data-testid="archive-completed">
			<ul class="flex flex-col gap-1">
				{#each completed as row (completedRowKey(row))}
					<li>
						<WorklistRow
							data-testid="archive-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							accentColor={row.jobColor}
							href={hrefOrgTarget(completedRowTarget(row))}
						>
							<WorklistJobLine
								mode="progress"
								size="sm"
								jobId={row.jobId}
								jobTitle={row.jobTitle}
								label={row.instanceLabel}
								done={row.tasksDone}
								total={row.tasksTotal}
								progressTestId="archive-progress"
							/>
						</WorklistRow>
					</li>
				{:else}
					<li>
						<WorklistRow data-testid="archive-empty" class="text-sm text-zinc-500">
							{m.no_completed_jobs()}
						</WorklistRow>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>
