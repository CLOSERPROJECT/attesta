<script lang="ts">
	import { untrack } from 'svelte';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import {
		getBoardTimelineModel,
		type VisibleStep
	} from '../board/board-timeline-model.svelte.js';
	import { displayStepTitle } from './outline-label.js';
	import { nestStatusChipVisible, streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineRow from './timeline-row.svelte';
	import TimelineSubstep from './timeline-substep.svelte';

	type Props = {
		projected: VisibleStep;
		stream?: string;
		instance?: string;
	};

	let { projected, stream = '', instance = '' }: Props = $props();

	const timelineModel = getBoardTimelineModel();

	const step = $derived(projected.step);
	const stepDomId = $derived(`stream-timeline-step-${step.stepId}`);
	const summaryDomId = $derived(`${stepDomId}-summary`);
	const stepMark = $derived(projected.mark);
	const title = $derived(displayStepTitle(step.title));
	const status = $derived(step.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');
	const showStatusChip = $derived(
		nestStatusChipVisible(status, {
			nestStatusChips: timelineModel.chrome.nestStatusChips,
			projectAll: timelineModel.chrome.projectAll
		})
	);
	const visibleRows = $derived(projected.rows ?? []);
	const visibleSubsteps = $derived(projected.substeps ?? []);

	let open = $state(Boolean(untrack(() => step.expanded)));
</script>

<Collapsible.Root class="stream-timeline-step" id={stepDomId} bind:open>
	<Collapsible.Trigger
		class="stream-timeline-step-summary w-full items-center hover:no-underline"
		id={summaryDomId}
		aria-labelledby={summaryDomId}
	>
		{#if showStatusChip}
			<TimelineHeaderLine mark={stepMark} {title}>
				{#snippet trail()}
					<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
				{/snippet}
			</TimelineHeaderLine>
		{:else}
			<TimelineHeaderLine mark={stepMark} {title} />
		{/if}
	</Collapsible.Trigger>
	<Collapsible.Content>
		{#if visibleRows.length > 0}
			<ul class="stream-timeline-rows">
				{#each visibleRows as entry (entry.row.id)}
					<li class="stream-timeline-row">
						<TimelineRow projected={entry} {stream} {instance} />
					</li>
				{/each}
			</ul>
		{:else}
			<ul class="stream-timeline-substeps">
				{#each visibleSubsteps as entry (entry.sub.substepId)}
					<TimelineSubstep
						substep={entry.sub}
						mark={entry.mark}
						{stream}
						{instance}
					/>
				{/each}
			</ul>
		{/if}
	</Collapsible.Content>
</Collapsible.Root>
