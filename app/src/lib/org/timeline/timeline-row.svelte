<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineSubstep from './timeline-substep.svelte';
	import type { TimelineRow } from './types.js';

	type Props = {
		row: TimelineRow;
		/** Hierarchical mark from parent step, e.g. 2A. */
		mark: string;
		stream?: string;
		instance?: string;
	};

	let { row, mark, stream = '', instance = '' }: Props = $props();

	let open = $state(Boolean(row.expanded));
</script>

<Collapsible.Root
	class="stream-timeline-row-accordion"
	id={`stream-timeline-row-${row.id.replaceAll('.', '-')}`}
	bind:open
>
	<Collapsible.Trigger class="stream-timeline-row-summary w-full items-center hover:no-underline">
		<TimelineHeaderLine {mark} title={row.label} />
	</Collapsible.Trigger>
	<Collapsible.Content>
		<ul class="stream-timeline-substeps">
			{#each row.substeps as sub, i (sub.substepId)}
				<TimelineSubstep
					substep={sub}
					mark={`${mark}.${i + 1}`}
					{stream}
					{instance}
				/>
			{/each}
		</ul>
	</Collapsible.Content>
</Collapsible.Root>
