<script lang="ts">
	import Inbox from '@lucide/svelte/icons/inbox';
	import * as m from '#lib/paraglide/messages.js';

	import ManySubmissionsTable from '../many-submissions-table.svelte';
	import { taskAddressKey, type TaskAddress } from '../task-address.js';
	import { boardJustCompleted } from './board-just-completed.svelte.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';
	import { pinSubmissionIdForAddress } from './just-completed-recap-row.js';

	type Props = {
		address: TaskAddress;
		/** Known N=0 — show empty without requiring a cache fetch. */
		empty?: boolean;
		'data-testid'?: string;
	};

	let {
		address,
		empty = false,
		'data-testid': testId = 'board-many-recap'
	}: Props = $props();

	const entry = $derived.by(() => {
		void boardRecapCache.entries;
		return boardRecapCache.get(taskAddressKey(address));
	});

	const highlightSubmissionId = $derived.by(() =>
		pinSubmissionIdForAddress(boardJustCompleted.get(), address)
	);
</script>

<div data-testid={testId}>
	{#if empty}
		<div
			class="flex items-center justify-center gap-2 rounded-md bg-muted/60 px-3 py-3 text-muted-foreground"
			data-testid="many-recap-empty"
		>
			<Inbox class="size-4 shrink-0" aria-hidden="true" />
			<p class="m-0 text-sm">{m.no_data_yet()}</p>
		</div>
	{:else if !entry || entry.status === 'idle' || entry.status === 'loading'}
		<p class="text-sm text-zinc-500" data-testid="many-recap-loading">
			{m.loading_recap()}
		</p>
	{:else if entry.status === 'error'}
		<p class="text-sm text-red-700" role="alert" data-testid="many-recap-error">
			{entry.error ?? m.could_not_load_recap()}
		</p>
	{:else if entry.table.rows.length === 0}
		<div
			class="flex items-center justify-center gap-2 rounded-md bg-muted/60 px-3 py-3 text-muted-foreground"
			data-testid="many-recap-empty"
		>
			<Inbox class="size-4 shrink-0" aria-hidden="true" />
			<p class="m-0 text-sm">{m.no_data_yet()}</p>
		</div>
	{:else}
		<ManySubmissionsTable table={entry.table} {highlightSubmissionId} />
	{/if}
</div>
