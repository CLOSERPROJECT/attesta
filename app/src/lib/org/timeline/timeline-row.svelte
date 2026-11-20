<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import TimelineSubstep from './timeline-substep.svelte';
	import type { TimelineRow } from './types.js';

	type Props = {
		row: TimelineRow;
		stream?: string;
		instance?: string;
	};

	let { row, stream = '', instance = '' }: Props = $props();

	let open = $state(false);

	$effect.pre(() => {
		if (row.expanded) open = true;
	});
</script>

<Collapsible.Root
	class="stream-timeline-row-accordion"
	id={`stream-timeline-row-${row.id.replaceAll('.', '-')}`}
	bind:open
>
	<Collapsible.Trigger class="stream-timeline-row-summary w-full items-center hover:no-underline">
		<span class="stream-timeline-row-label">{row.label}</span>
	</Collapsible.Trigger>
	<Collapsible.Content>
		<ul class="stream-timeline-substeps">
			{#each row.substeps as sub (sub.substepId)}
				<TimelineSubstep substep={sub} {stream} {instance} />
			{/each}
		</ul>
	</Collapsible.Content>
</Collapsible.Root>
