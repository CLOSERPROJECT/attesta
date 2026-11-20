<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineSubstep from './timeline-substep.svelte';
	import type { TimelineRow } from './types.js';

	type Props = {
		row: TimelineRow;
		stream?: string;
		instance?: string;
	};

	let { row, stream = '', instance = '' }: Props = $props();

	let open = $state(Boolean(row.expanded));
</script>

<Collapsible.Root
	class="stream-timeline-row-accordion"
	id={`stream-timeline-row-${row.id.replaceAll('.', '-')}`}
	bind:open
>
	<Collapsible.Trigger class="stream-timeline-row-summary w-full items-center hover:no-underline">
		<TimelineHeaderLine title={row.label} />
	</Collapsible.Trigger>
	<Collapsible.Content>
		<ul class="stream-timeline-substeps">
			{#each row.substeps as sub (sub.substepId)}
				<TimelineSubstep substep={sub} {stream} {instance} />
			{/each}
		</ul>
	</Collapsible.Content>
</Collapsible.Root>
