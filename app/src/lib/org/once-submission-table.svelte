<script lang="ts">
	import HttpLinkStrip from './form/http-link-strip.svelte';
	import {
		formatSubmissionTimestamp,
		formatSubmissionUser
	} from './submission-table.js';
	import type { TaskRecap } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		item: TaskRecap;
		'data-testid'?: string;
	};

	let { item, 'data-testid': testId = 'once-submission-table' }: Props = $props();

	const factEntries = $derived(
		Object.entries(item.facts).filter(([k]) => k.length > 0)
	);
	const titles = $derived(item.titles ?? {});
	const timestamp = $derived(formatSubmissionTimestamp(item.timestamp));
	const user = $derived(formatSubmissionUser(item));
</script>

<div class="space-y-3" data-testid={testId}>
	<table class="w-full border-collapse text-sm">
		<tbody>
			{#each factEntries as [key, value] (key)}
				<tr class="border-b border-zinc-100">
					<th class="py-2 pr-3 text-left font-medium text-zinc-600">
						{titles[key] ?? key}
					</th>
					<td class="py-2 text-zinc-900">{formatFactValue(value)}</td>
				</tr>
			{/each}
			<tr class="border-b border-zinc-100">
				<th class="py-2 pr-3 text-left font-medium text-zinc-600">Timestamp</th>
				<td class="py-2 text-zinc-900">{timestamp}</td>
			</tr>
			<tr class="border-b border-zinc-100 last:border-0">
				<th class="py-2 pr-3 text-left font-medium text-zinc-600">User</th>
				<td class="py-2 text-zinc-900">{user}</td>
			</tr>
		</tbody>
	</table>
	{#if item.links.length > 0}
		<HttpLinkStrip items={item.links} />
	{/if}
</div>
