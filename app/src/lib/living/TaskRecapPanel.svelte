<script lang="ts">
	import type { HttpLinkItem } from './commit-port.js';
	import HttpLinkStrip from './form/HttpLinkStrip.svelte';
	import { createHttpTaskRecapPort } from './http-task-recap-port.js';
	import { TaskRecapError, type TaskRecapRequest } from './task-recap-port.js';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		request: TaskRecapRequest;
		/** When false, skip fetch until the accordion opens. */
		active?: boolean;
		'data-testid'?: string;
	};

	let { request, active = true, 'data-testid': testId = 'overview-task-recap' }: Props =
		$props();

	const port = createHttpTaskRecapPort();

	let loading = $state(false);
	let error = $state<string | null>(null);
	let facts = $state<Record<string, unknown> | null>(null);
	let links = $state<HttpLinkItem[]>([]);
	let fetchedKey = $state('');

	const requestKey = $derived(
		[
			request.jobId,
			request.jobInstanceId,
			request.taskId,
			request.groupId ?? '',
			request.groupInstanceId ?? ''
		].join(':')
	);

	$effect(() => {
		if (!active) return;
		const key = requestKey;
		if (fetchedKey === key) return;
		fetchedKey = key;
		loading = true;
		error = null;
		facts = null;
		links = [];
		void port
			.fetch(request)
			.then((recap) => {
				facts = recap.facts;
				links = recap.links;
			})
			.catch((err: unknown) => {
				error =
					err instanceof TaskRecapError
						? err.message
						: err instanceof Error
							? err.message
							: 'Could not load recap';
			})
			.finally(() => {
				loading = false;
			});
	});

	const factEntries = $derived(
		facts ? Object.entries(facts).filter(([k]) => k.length > 0) : []
	);
</script>

<div class="space-y-3 px-1" data-testid={testId}>
	{#if loading}
		<p class="text-sm text-zinc-500">Loading recap…</p>
	{:else if error}
		<p class="text-sm text-red-700" role="alert">{error}</p>
	{:else if facts}
		<dl class="grid gap-2 text-sm" data-testid="submission-summary">
			{#each factEntries as [key, value] (key)}
				<div class="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-2 border-b border-zinc-100 pb-2 last:border-0">
					<dt class="font-medium text-zinc-600">{key}</dt>
					<dd class="text-zinc-900">{formatFactValue(value)}</dd>
				</div>
			{/each}
		</dl>
		<HttpLinkStrip items={links} />
	{/if}
</div>
