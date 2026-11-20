<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineRow from './timeline-row.svelte';
	import TimelineSubstep from './timeline-substep.svelte';
	import type { TimelineStep } from './types.js';

	type Props = {
		step: TimelineStep;
		stream?: string;
		instance?: string;
	};

	let { step, stream = '', instance = '' }: Props = $props();

	const stepDomId = $derived(`stream-timeline-step-${step.stepId}`);
	const summaryDomId = $derived(`${stepDomId}-summary`);
	const rows = $derived(step.rows ?? []);
	const substeps = $derived(step.substeps ?? []);

	let open = $state(Boolean(step.expanded));
</script>

<Collapsible.Root class="stream-timeline-step" id={stepDomId} bind:open>
	<Collapsible.Trigger
		class="stream-timeline-step-summary w-full items-center hover:no-underline"
		id={summaryDomId}
		aria-labelledby={summaryDomId}
	>
		<TimelineHeaderLine lead={step.stepId} title={step.title}>
			{#snippet trail()}
				{#if step.substepCount > 0}
					<span class="timeline-header-count">{step.substepCount}</span>
				{/if}
			{/snippet}
		</TimelineHeaderLine>
	</Collapsible.Trigger>
	<Collapsible.Content>
		{#if rows.length > 0}
			<ul class="stream-timeline-rows">
				{#each rows as row (row.id)}
					<li class="stream-timeline-row">
						<TimelineRow {row} {stream} {instance} />
					</li>
				{/each}
			</ul>
		{:else}
			<ul class="stream-timeline-substeps">
				{#each substeps as sub (sub.substepId)}
					<TimelineSubstep substep={sub} {stream} {instance} />
				{/each}
			</ul>
		{/if}
	</Collapsible.Content>
</Collapsible.Root>
