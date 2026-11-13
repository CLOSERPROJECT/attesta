<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';

	import type { HttpLinkItem } from './commit-port.js';
	import TaskRecapFacts from './TaskRecapFacts.svelte';
	import { createHttpTaskRecapPort } from './http-task-recap-port.js';
	import { TaskRecapError, type TaskRecapRequest } from './task-recap-port.js';

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
	let error = $state.raw<string | null>(null);
	let facts = $state.raw<Record<string, unknown> | null>(null);
	let titles = $state.raw<Record<string, string>>({});
	let links = $state.raw<HttpLinkItem[]>([]);
	let fetchedKey = $state.raw('');

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
		titles = {};
		links = [];
		void port
			.fetch(request)
			.then((list) => {
				const recap = list.items[0];
				if (!recap) return;
				facts = recap.facts;
				titles = recap.titles ?? {};
				links = recap.links;
			})
			.catch((err: unknown) => {
				error =
					err instanceof TaskRecapError
						? err.message
						: err instanceof Error
							? err.message
							: m.could_not_load_recap();
			})
			.finally(() => {
				loading = false;
			});
	});
</script>

<div class="space-y-3 px-1" data-testid={testId}>
	{#if loading}
		<p class="text-sm text-zinc-500">{m.loading_recap()}</p>
	{:else if error}
		<p class="text-sm text-red-700" role="alert">{error}</p>
	{:else if facts}
		<TaskRecapFacts {facts} {titles} {links} />
	{/if}
</div>
