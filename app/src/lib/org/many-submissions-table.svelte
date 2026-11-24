<script lang="ts">
	import HttpLinkStrip from './form/http-link-strip.svelte';
	import {
		factTitle,
		formatSubmissionTimestamp,
		formatSubmissionUser,
		recapRowMatchesSubmissionId,
		unionFactKeys
	} from './submission-table.js';
	import type { TaskRecap } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		items: TaskRecap[];
		/** Just-completed Commit submissionId (Submission.timestamp) to highlight. */
		highlightSubmissionId?: string;
		'data-testid'?: string;
	};

	let {
		items,
		highlightSubmissionId,
		'data-testid': testId = 'many-submissions-table'
	}: Props = $props();

	const columns = $derived(unionFactKeys(items));
	const showLinksColumn = $derived(items.some((item) => item.links.length > 0));
</script>

{#if items.length > 0}
	<div class="overflow-x-auto" data-testid={testId}>
		<table class="w-full border-collapse border-y border-zinc-200 text-sm">
			<thead>
				<tr class="border-b border-zinc-200 text-left">
					{#each columns as key (key)}
						<th class="px-2 py-2 align-top font-medium text-zinc-600">
							{factTitle(items, key)}
						</th>
					{/each}
					<th class="px-2 py-2 align-top font-medium text-zinc-600">Timestamp</th>
					<th class="px-2 py-2 align-top font-medium text-zinc-600">User</th>
					{#if showLinksColumn}
						<th class="px-2 py-2 align-top font-medium text-zinc-600">Links</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each items as item, index (`${item.timestamp}:${item.jobInstanceId}:${index}`)}
					{@const highlighted = recapRowMatchesSubmissionId(
						item,
						highlightSubmissionId
					)}
					<tr
						class={[
							'border-b border-zinc-100 last:border-0',
							highlighted && 'bg-sky-50'
						]}
						data-just-completed={highlighted ? '1' : undefined}
						data-testid={highlighted ? 'many-recap-just-completed' : undefined}
					>
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
						{#if showLinksColumn}
							<td class="px-2 py-2">
								{#if item.links.length > 0}
									<HttpLinkStrip items={item.links} />
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
