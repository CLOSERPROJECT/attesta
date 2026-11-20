<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import { displayStepTitle, outlineLetter } from './outline-label.js';
	import { streamStatusLabel } from './stream-status-label.js';
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

	const nestCollapse = getBoardNestCollapse();

	const stepDomId = $derived(`stream-timeline-step-${step.stepId}`);
	const summaryDomId = $derived(`${stepDomId}-summary`);
	const rows = $derived(step.rows ?? []);
	const substeps = $derived(step.substeps ?? []);
	const mark = $derived(step.stepId);
	const title = $derived(displayStepTitle(step.title));
	const status = $derived(step.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');

	let open = $state(Boolean(step.expanded));
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		open = false;
	});
</script>

<Collapsible.Root class="stream-timeline-step" id={stepDomId} bind:open>
	<Collapsible.Trigger
		class="stream-timeline-step-summary w-full items-center hover:no-underline"
		id={summaryDomId}
		aria-labelledby={summaryDomId}
	>
		{#if status}
			<TimelineHeaderLine {mark} {title}>
				{#snippet trail()}
					<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
				{/snippet}
			</TimelineHeaderLine>
		{:else}
			<TimelineHeaderLine {mark} {title} />
		{/if}
	</Collapsible.Trigger>
	<Collapsible.Content>
		{#if rows.length > 0}
			<ul class="stream-timeline-rows">
				{#each rows as row, i (row.id)}
					<li class="stream-timeline-row">
						<TimelineRow
							{row}
							mark={`${mark}${outlineLetter(i)}`}
							{stream}
							{instance}
						/>
					</li>
				{/each}
			</ul>
		{:else}
			<ul class="stream-timeline-substeps">
				{#each substeps as sub, i (sub.substepId)}
					<TimelineSubstep
						substep={sub}
						mark={`${mark}.${i + 1}`}
						{stream}
						{instance}
					/>
				{/each}
			</ul>
		{/if}
	</Collapsible.Content>
</Collapsible.Root>
