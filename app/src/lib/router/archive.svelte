<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import { living } from '../living/living-app.svelte.js';
	import WorklistJobLine from '../living/WorklistJobLine.svelte';
	import WorklistRow from '../living/WorklistRow.svelte';
	import {
		completedRowKey,
		completedRowsFromWorklist,
		completedRowTarget
	} from '../living/worklist-home.js';
	import { hrefLivingTarget } from './navigate-target.js';

	onMount(() => {
		if (!living.result && !living.loading) {
			void living.hydrate();
		}
	});

	const completed = $derived(completedRowsFromWorklist(living.result?.worklist));
</script>

<div class="mx-auto max-w-3xl space-y-6" data-testid="archive-route">
	{#if !living.result && living.loading}
		<p class="text-sm text-zinc-500" data-testid="archive-load-status">{m.loading()}</p>
	{:else if !living.result && living.error}
		<p class="text-sm text-red-700" data-testid="archive-load-error" role="alert">
			{living.error}
		</p>
	{:else if !living.result}
		<p class="text-sm text-zinc-500" data-testid="archive-load-status">
			{m.not_loaded_yet()}
		</p>
	{/if}

	{#if living.result}
		<section class="space-y-3" data-testid="archive-completed">
			<ul class="flex flex-col gap-1">
				{#each completed as row (completedRowKey(row))}
					<li>
						<WorklistRow
							data-testid="archive-row"
							data-job-id={row.jobId}
							data-instance-id={row.jobInstanceId}
							accentColor={row.jobColor}
							href={hrefLivingTarget(completedRowTarget(row))}
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
