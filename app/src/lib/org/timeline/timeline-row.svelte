<script lang="ts">
	import { untrack } from 'svelte';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import type { VisibleRow } from '../board/board-timeline-model.svelte.js';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import { streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineSubstep from './timeline-substep.svelte';

	type Props = {
		projected: VisibleRow;
		stream?: string;
		instance?: string;
	};

	let { projected, stream = '', instance = '' }: Props = $props();

	const nestCollapse = getBoardNestCollapse();

	const row = $derived(projected.row);
	const mark = $derived(projected.mark);
	const status = $derived(row.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');
	const visibleSubsteps = $derived(projected.substeps);

	let open = $state(Boolean(untrack(() => row.expanded)));
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		open = false;
	});
</script>

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
