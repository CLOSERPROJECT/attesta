<script lang="ts">
	import { untrack } from 'svelte';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import { timelineSubstepShown } from '../board/timeline-actionable.js';
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
		showAll?: boolean;
	};

	let { step, stream = '', instance = '', showAll = false }: Props = $props();

	const board = getBoardEmbed();
	const nestCollapse = getBoardNestCollapse();

	const stepDomId = $derived(`stream-timeline-step-${step.stepId}`);
	const summaryDomId = $derived(`${stepDomId}-summary`);
	const stepMark = $derived(step.stepId);
	const title = $derived(displayStepTitle(step.title));
	const status = $derived(step.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');

	const filterOpts = $derived({
		showAll,
		focus: board.focus,
		stream,
		instance
	});

	const visibleRows = $derived(
		(step.rows ?? [])
			.map((row, i) => ({
				row,
				mark: `${stepMark}${outlineLetter(i)}`
			}))
			.filter(({ row }) =>
				row.substeps.some((sub) => timelineSubstepShown(sub, filterOpts))
			)
	);

	const visibleSubsteps = $derived(
		(step.substeps ?? [])
			.map((sub, i) => ({
				sub,
				mark: `${stepMark}.${i + 1}`
			}))
			.filter(({ sub }) => timelineSubstepShown(sub, filterOpts))
	);

	let open = $state(Boolean(untrack(() => step.expanded)));
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		open = false;
	});
</script>

{#if showAll || visibleRows.length > 0 || visibleSubsteps.length > 0}
	<Collapsible.Root class="stream-timeline-step" id={stepDomId} bind:open>
		<Collapsible.Trigger
			class="stream-timeline-step-summary w-full items-center hover:no-underline"
			id={summaryDomId}
			aria-labelledby={summaryDomId}
		>
			{#if status}
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
			{#if (step.rows ?? []).length > 0}
				<ul class="stream-timeline-rows">
					{#each visibleRows as entry (entry.row.id)}
						<li class="stream-timeline-row">
							<TimelineRow
								row={entry.row}
								mark={entry.mark}
								{showAll}
								{stream}
								{instance}
							/>
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
{/if}
