<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';

	import OnceSubmissionTable from '../once-submission-table.svelte';
	import { taskAddressKey, type TaskAddress } from '../task-address.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';

	type Props = {
		address: TaskAddress;
		'data-testid'?: string;
	};

	let { address, 'data-testid': testId }: Props = $props();

	const entry = $derived.by(() => {
		void boardRecapCache.entries;
		return boardRecapCache.get(taskAddressKey(address));
	});
</script>

{#if !entry || entry.status === 'idle' || entry.status === 'loading'}
	<p class="text-sm text-zinc-500" data-testid="once-recap-loading">{m.loading_recap()}</p>
{:else if entry.status === 'error'}
	<p class="text-sm text-red-700" role="alert" data-testid="once-recap-error">
		{entry.error ?? m.could_not_load_recap()}
	</p>
{:else if entry.table.rows.length > 0}
	<OnceSubmissionTable
		columns={entry.table.columns}
		row={entry.table.rows[0]}
		data-testid={testId}
	/>
{/if}
