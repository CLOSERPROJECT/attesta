<script lang="ts">
	import HttpLinkStrip from './form/http-link-strip.svelte';
	import {
		factTitle,
		formatSubmissionTimestamp,
		formatSubmissionUser,
		unionFactKeys
	} from './submission-table.js';
	import type { TaskRecap } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		items: TaskRecap[];
		'data-testid'?: string;
	};

	let { items, 'data-testid': testId = 'many-submissions-table' }: Props = $props();

	const columns = $derived(unionFactKeys(items));
</script>

{#if items.length > 0}
	<div class="overflow-x-auto" data-testid={testId}>
		<table class="w-full border-collapse text-sm">
			<thead>
				<tr class="border-b border-zinc-200 text-left">
					{#each columns as key (key)}
						<th class="px-2 py-2 align-top font-medium text-zinc-600">
							{factTitle(items, key)}
						</th>
					{/each}
					<th class="px-2 py-2 align-top font-medium text-zinc-600">Timestamp</th>
					<th class="px-2 py-2 align-top font-medium text-zinc-600">User</th>
					<th class="px-2 py-2 align-top font-medium text-zinc-600">Links</th>
				</tr>
			</thead>
			<tbody>
				{#each items as item, index (`${item.timestamp}:${item.jobInstanceId}:${index}`)}
					<tr class="border-b border-zinc-100 last:border-0">
						{#each columns as key (key)}
							<td class="px-2 py-2 text-zinc-900">
								{formatFactValue(item.facts[key])}
							</td>
						{/each}
						<td class="px-2 py-2 text-zinc-900">
							{formatSubmissionTimestamp(item.timestamp)}
						</td>
						<td class="px-2 py-2 text-zinc-900">
							{formatSubmissionUser(item)}
						</td>
						<td class="px-2 py-2">
							{#if item.links.length > 0}
								<HttpLinkStrip items={item.links} />
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
