<script lang="ts">
	import HttpLinkStrip from './form/http-link-strip.svelte';
	import {
		formatSubmissionTimestamp,
		formatSubmissionUser,
		recapRowMatchesSubmissionId
	} from './submission-table.js';
	import type { TaskRecapTable } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		table: TaskRecapTable;
		/** Just-completed Commit submissionId (Submission.timestamp) to highlight. */
		highlightSubmissionId?: string;
		'data-testid'?: string;
	};

	let {
		table,
		highlightSubmissionId,
		'data-testid': testId = 'many-submissions-table'
	}: Props = $props();

	const showLinksColumn = $derived(
		table.rows.some((row) => (row.links?.length ?? 0) > 0)
	);
	/** Fact cols + Timestamp + User + optional Links. */
	const colCount = $derived(table.columns.length + 2 + (showLinksColumn ? 1 : 0));

	function edgePad(index: number): string {
		const pl = index === 0 ? 'pl-0' : 'pl-2';
		const pr = index === colCount - 1 ? 'pr-0' : 'pr-2';
		return `${pl} ${pr}`;
	}
</script>

{#if table.rows.length > 0}
	<div class="overflow-x-auto" data-testid={testId}>
		<table class="w-full border-collapse border-t border-zinc-200 text-sm">
			<thead>
				<tr class="border-b border-zinc-200 text-left">
					{#each table.columns as col, i (col.key)}
						<th class="{edgePad(i)} py-2 align-top font-semibold text-muted-foreground">
							{col.title}
						</th>
					{/each}
					<th
						class="{edgePad(table.columns.length)} py-2 align-top font-semibold text-muted-foreground"
					>
						Timestamp
					</th>
					<th
						class="{edgePad(table.columns.length + 1)} py-2 align-top font-semibold text-muted-foreground"
					>
						User
					</th>
					{#if showLinksColumn}
						<th
							class="{edgePad(table.columns.length + 2)} py-2 align-top font-semibold text-muted-foreground"
						>
							Links
						</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each table.rows as row (row.key)}
					{@const highlighted = recapRowMatchesSubmissionId(
						row,
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
						{#each table.columns as col, i (col.key)}
							<td class="{edgePad(i)} py-2 font-normal text-muted-foreground">
								{formatFactValue(row.cells[col.key])}
							</td>
						{/each}
						<td class="{edgePad(table.columns.length)} py-2 font-normal text-muted-foreground">
							{formatSubmissionTimestamp(row.timestamp)}
						</td>
						<td
							class="{edgePad(table.columns.length + 1)} py-2 font-normal text-muted-foreground"
						>
							{formatSubmissionUser(row)}
						</td>
						{#if showLinksColumn}
							<td class="{edgePad(table.columns.length + 2)} py-2 font-normal">
								{#if (row.links?.length ?? 0) > 0}
									<HttpLinkStrip items={row.links ?? []} />
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
