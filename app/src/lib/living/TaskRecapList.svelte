<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';

	import TaskRecapFacts from './TaskRecapFacts.svelte';
	import { createHttpTaskRecapPort } from './http-task-recap-port.js';
	import {
		TaskRecapError,
		type TaskRecap,
		type TaskRecapRequest
	} from './task-recap-port.js';

	type Props = {
		request: TaskRecapRequest;
		/** When false, skip fetch until the accordion opens. */
		active?: boolean;
		'data-testid'?: string;
	};

	let {
		request,
		active = true,
		'data-testid': testId = 'overview-task-recap-list'
	}: Props = $props();

	const port = createHttpTaskRecapPort();

	let loading = $state(false);
	let error = $state.raw<string | null>(null);
	let items = $state.raw<TaskRecap[]>([]);
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
		items = [];
		void port
			.fetch(request)
			.then((list) => {
				items = list.items;
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

<div class="space-y-4 px-1" data-testid={testId}>
	{#if loading}
		<p class="text-sm text-zinc-500">{m.loading_recap()}</p>
	{:else if error}
		<p class="text-sm text-red-700" role="alert">{error}</p>
	{:else}
		{#each items as item, index (item.timestamp || index)}
			<div
				class="space-y-3 border-b border-zinc-100 pb-4 last:border-0 last:pb-0"
				data-testid="overview-task-recap-item"
				data-recap-index={index}
			>
				<TaskRecapFacts
					facts={item.facts}
					titles={item.titles ?? {}}
					links={item.links}
				/>
			</div>
		{/each}
	{/if}
</div>
