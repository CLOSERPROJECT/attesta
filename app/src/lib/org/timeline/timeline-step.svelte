<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

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

	let open = $state(false);

	$effect.pre(() => {
		if (step.expanded) open = true;
	});
</script>

<Collapsible.Root class="stream-timeline-step" id={stepDomId} bind:open>
	<Collapsible.Trigger
		class="stream-timeline-step-summary w-full items-center hover:no-underline"
		id={summaryDomId}
		aria-labelledby={summaryDomId}
	>
		<div class="stream-step-summary-main">
			<span class="pill pill-step-number pill-panel">{step.stepId}</span>
			<span class="stream-step-summary-copy">
				<span class="stream-step-summary-title">{step.title}</span>
				{#if step.substepCount}
					<div class="stream-step-summary-meta">
						<span class="stream-step-summary-meta-item">{step.substepCount}</span>
					</div>
				{/if}
			</span>
		</div>
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
