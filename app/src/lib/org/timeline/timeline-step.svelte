<script lang="ts">
	import TimelineChevron from './timeline-chevron.svelte';
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
</script>

<details class="stream-timeline-step" id={stepDomId} open={step.expanded || undefined}>
	<summary class="stream-timeline-step-summary" id={summaryDomId} aria-labelledby={summaryDomId}>
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
		<span class="stream-timeline-step-chevron" aria-hidden="true">
			<TimelineChevron />
		</span>
	</summary>
	{#if rows.length > 0}
		<ul class="stream-timeline-rows">
			{#each rows as row (row.id)}
				<li class="stream-timeline-row">
					<details
						class="stream-timeline-row-accordion"
						id={`stream-timeline-row-${row.id.replaceAll('.', '-')}`}
						open={row.expanded || undefined}
					>
						<summary class="stream-timeline-row-summary">
							<span class="stream-timeline-row-label">{row.label}</span>
							<span class="stream-timeline-row-chevron" aria-hidden="true">
								<TimelineChevron />
							</span>
						</summary>
						<ul class="stream-timeline-substeps">
							{#each row.substeps as sub (sub.substepId)}
								<TimelineSubstep substep={sub} {stream} {instance} />
							{/each}
						</ul>
					</details>
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
</details>
