<script lang="ts">
	import { untrack } from 'svelte';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import { timelineSubstepShown } from '../board/timeline-actionable.js';
	import { streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineSubstep from './timeline-substep.svelte';
	import type { TimelineRow } from './types.js';

	type Props = {
		row: TimelineRow;
		/** Hierarchical mark from parent step, e.g. 2A. */
		mark: string;
		stream?: string;
		instance?: string;
		showAll?: boolean;
	};

	let { row, mark, stream = '', instance = '', showAll = false }: Props = $props();

	const board = getBoardEmbed();
	const nestCollapse = getBoardNestCollapse();

	const status = $derived(row.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');

	const filterOpts = $derived({
		showAll,
		focus: board.focus,
		stream,
		instance
	});

	const visibleSubsteps = $derived(
		row.substeps
			.map((sub, i) => ({
				sub,
				mark: `${mark}.${i + 1}`
			}))
			.filter(({ sub }) => timelineSubstepShown(sub, filterOpts))
	);

	let open = $state(Boolean(untrack(() => row.expanded)));
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		open = false;
	});
</script>

{#if showAll || visibleSubsteps.length > 0}
	<Collapsible.Root
		class="stream-timeline-row-accordion"
		id={`stream-timeline-row-${row.id.replaceAll('.', '-')}`}
		bind:open
	>
		<Collapsible.Trigger class="stream-timeline-row-summary w-full items-center hover:no-underline">
			{#if status}
				<TimelineHeaderLine {mark} title={row.label}>
					{#snippet trail()}
						<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
					{/snippet}
				</TimelineHeaderLine>
			{:else}
				<TimelineHeaderLine {mark} title={row.label} />
			{/if}
		</Collapsible.Trigger>
		<Collapsible.Content>
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
		</Collapsible.Content>
	</Collapsible.Root>
{/if}
