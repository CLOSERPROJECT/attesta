<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import * as m from '#lib/paraglide/messages.js';

	import type { TaskAddress } from '../task-address.js';
	import BoardManyRecap from './board-many-recap.svelte';

	type Props = {
		address: TaskAddress;
		bandTestId?: string;
		recapTestId?: string;
	};

	let {
		address,
		bandTestId = 'embed-many-recap-band',
		recapTestId = 'embed-many-recap'
	}: Props = $props();

	const uid = $props.id();
	const eyebrowId = `${uid}-many-recap-eyebrow`;
</script>

<section class="embed-many-recap-band bg-muted/60" data-testid={bandTestId} aria-labelledby={eyebrowId}>
	<p
		id={eyebrowId}
		class="m-0 flex items-center gap-1.5 text-sm font-bold text-muted-foreground"
		data-testid="embed-many-recap-eyebrow"
	>
		<CheckIcon class="size-3.5 shrink-0" aria-hidden="true" />
		{m.embed_many_already_registered()}
	</p>
	<BoardManyRecap {address} empty={false} data-testid={recapTestId} />
</section>
