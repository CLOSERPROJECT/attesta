<script lang="ts">
	import type { Action } from 'svelte/action';

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

	/** Hold full emphasis, then CSS fade (must match --many-recap-wash-ms). */
	const HIGHLIGHT_HOLD_MS = 4000;
	const HIGHLIGHT_FADE_MS = 700;

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

	/** Local visual wash — pin may remain for timeline `has()`. */
	let phase: 'off' | 'on' | 'fade' = $state('off');
	let barTop = $state(0);
	let barHeight = $state(0);

	const showBar = $derived(
		Boolean(highlightSubmissionId) && phase !== 'off' && barHeight > 0
	);

	$effect(() => {
		const id = highlightSubmissionId;
		if (!id) {
			phase = 'off';
			return;
		}
		phase = 'on';
		const fadeAt = setTimeout(() => {
			phase = 'fade';
		}, HIGHLIGHT_HOLD_MS);
		const offAt = setTimeout(() => {
			phase = 'off';
		}, HIGHLIGHT_HOLD_MS + HIGHLIGHT_FADE_MS);
		return () => {
			clearTimeout(fadeAt);
			clearTimeout(offAt);
		};
	});

	const trackHighlightRow: Action<HTMLTableRowElement, boolean> = (node, active) => {
		let observer: ResizeObserver | undefined;

		function sync() {
			const host = node.closest('.many-recap-highlight-host');
			if (!(host instanceof HTMLElement)) return;
			const hostRect = host.getBoundingClientRect();
			const rowRect = node.getBoundingClientRect();
			barTop = rowRect.top - hostRect.top;
			barHeight = rowRect.height;
		}

		function teardown() {
			observer?.disconnect();
			observer = undefined;
			window.removeEventListener('resize', sync);
		}

		function setup(isActive: boolean) {
			teardown();
			if (!isActive) return;
			sync();
			observer = new ResizeObserver(sync);
			observer.observe(node);
			const tableEl = node.closest('table');
			if (tableEl) observer.observe(tableEl);
			window.addEventListener('resize', sync);
		}

		setup(active);
		return {
			update(isActive: boolean) {
				setup(isActive);
			},
			destroy() {
				teardown();
			}
		};
	};

	function edgePad(index: number): string {
		const pl = index === 0 ? 'pl-0' : 'pl-2';
		const pr = index === colCount - 1 ? 'pr-0' : 'pr-2';
		return `${pl} ${pr}`;
	}
</script>

{#if table.rows.length > 0}
	<div class="many-recap-highlight-host" data-testid={testId}>
		{#if showBar}
			<div
				class="many-recap-just-completed-bar"
				class:many-recap-just-completed-bar--fade={phase === 'fade'}
				aria-hidden="true"
				style:top="{barTop}px"
				style:height="{barHeight}px"
			></div>
		{/if}
		<div class="many-recap-table-scroll min-w-0 overflow-x-auto">
			<table class="w-max min-w-full border-collapse border-t border-border text-sm">
				<thead>
					<tr class="border-b border-border text-left">
						{#each table.columns as col, i (col.key)}
							<th
								class="{edgePad(i)} whitespace-nowrap py-2 align-top font-semibold text-muted-foreground"
							>
								{col.title}
							</th>
						{/each}
						<th
							class="{edgePad(table.columns.length)} whitespace-nowrap py-2 align-top font-semibold text-muted-foreground"
						>
							Timestamp
						</th>
						<th
							class="{edgePad(table.columns.length + 1)} whitespace-nowrap py-2 align-top font-semibold text-muted-foreground"
						>
							User
						</th>
						{#if showLinksColumn}
							<th
								class="{edgePad(table.columns.length + 2)} whitespace-nowrap py-2 align-top font-semibold text-muted-foreground"
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
						{@const track = highlighted && phase !== 'off'}
						{@const emphasize = highlighted && phase === 'on'}
						<tr
							class="border-b border-border last:border-0"
							use:trackHighlightRow={track}
							data-just-completed={emphasize ? '1' : undefined}
							data-just-completed-fade={highlighted && phase === 'fade'
								? '1'
								: undefined}
							data-testid={emphasize || (highlighted && phase === 'fade')
								? 'many-recap-just-completed'
								: undefined}
						>
							{#each table.columns as col, i (col.key)}
								<td
									class="{edgePad(i)} many-recap-cell whitespace-nowrap py-2 font-normal"
								>
									{formatFactValue(row.cells[col.key])}
								</td>
							{/each}
							<td
								class="{edgePad(table.columns.length)} many-recap-cell whitespace-nowrap py-2 font-normal"
							>
								{formatSubmissionTimestamp(row.timestamp)}
							</td>
							<td
								class="{edgePad(table.columns.length + 1)} many-recap-cell whitespace-nowrap py-2 font-normal"
							>
								{formatSubmissionUser(row)}
							</td>
							{#if showLinksColumn}
								<td
									class="{edgePad(table.columns.length + 2)} whitespace-nowrap py-2 font-normal"
								>
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
	</div>
{/if}
