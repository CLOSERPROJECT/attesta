<script lang="ts">
	import HttpLinkStrip from './form/http-link-strip.svelte';
	import {
		formatSubmissionTimestamp,
		formatSubmissionUser
	} from './submission-table.js';
	import type { TaskRecapCol, TaskRecapRow } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		columns: TaskRecapCol[];
		row: TaskRecapRow;
		'data-testid'?: string;
	};

	let { columns, row, 'data-testid': testId = 'once-submission-table' }: Props =
		$props();

	const timestamp = $derived(formatSubmissionTimestamp(row.timestamp));
	const user = $derived(formatSubmissionUser(row));
</script>

<div class="space-y-3" data-testid={testId}>
	<table class="w-full border-collapse text-sm">
		<tbody>
			{#each columns as column (column.key)}
				<tr class="border-b border-border">
					<th class="py-2 pr-3 text-left font-medium text-secondary-foreground">
						{column.title}
					</th>
					<td class="py-2 text-foreground">
						{formatFactValue(row.cells[column.key])}
					</td>
				</tr>
			{/each}
			<tr class="border-b border-border">
				<th class="py-2 pr-3 text-left font-medium text-secondary-foreground">Timestamp</th>
				<td class="py-2 text-foreground">{timestamp}</td>
			</tr>
			<tr class="border-b border-border last:border-0">
				<th class="py-2 pr-3 text-left font-medium text-secondary-foreground">User</th>
				<td class="py-2 text-foreground">{user}</td>
			</tr>
		</tbody>
	</table>
	{#if (row.links?.length ?? 0) > 0}
		<HttpLinkStrip items={row.links ?? []} />
	{/if}
</div>
