<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../org/commit-port.js';
	import {
		applyCommitFollowThrough,
		decideCommitFollowThrough
	} from '../org/commit-follow-through.js';
	import { createHttpTaskFormPort } from '../org/http-task-form-port.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { orgSession } from '../org/org-app.svelte.js';
	import { applySeedQueryParams } from '../org/seed-query.js';
	import TaskForm from '../org/task-form.svelte';
	import { normalizeTaskAddress, taskAddressKey } from '../org/task-address.js';
	import type { TaskFormBundle } from '../org/types.js';
	import { navigateOrgTarget, orgMyStreamDashboardPath } from './navigate-target.js';
	import { route } from './index.js';

	type StreamListItem = {
		key: string;
		title: string;
		play?: { jobId: string; taskId: string } | null;
	};

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const stream = $derived(coords.stream);

	let bundle = $state.raw<TaskFormBundle | null>(null);
	let playJob = $state.raw<{ jobId: string; taskId: string } | null>(null);
	let streamTitle = $state('');
	let error = $state.raw<string | null>(null);
	let loading = $state(true);

	const addr = $derived(
		playJob
			? normalizeTaskAddress({
					stream,
					instance: '',
					jobId: playJob.jobId,
					jobInstanceId: '',
					taskId: playJob.taskId
				})
			: null
	);
	const routeKey = $derived(addr ? taskAddressKey(addr) : '');
	const dashboardHref = $derived(orgMyStreamDashboardPath(stream));
	const titleText = $derived(streamTitle || stream);

	onMount(() => {
		let cancelled = false;

		void (async () => {
			loading = true;
			error = null;
			bundle = null;
			playJob = null;

			if (!stream) {
				error = 'stream missing';
				loading = false;
				return;
			}

			try {
				const res = await fetch('/api/org/streams', {
					credentials: 'include',
					headers: { Accept: 'application/json' },
					cache: 'no-store'
				});
				if (!res.ok) {
					throw new Error(await res.text());
				}
				const items = (await res.json()) as StreamListItem[];
				const row = items.find((item) => item.key === stream);
				const anchor = row?.play;
				if (!anchor?.jobId || !anchor.taskId) {
					throw new Error('play start not found');
				}
				if (cancelled) return;

				streamTitle = row?.title?.trim() || stream;

				const stripped = applySeedQueryParams(window.location.href, orgSession, {
					jobId: anchor.jobId,
					taskId: anchor.taskId,
					jobInstanceId: ''
				});
				if (stripped) {
					window.history.replaceState(window.history.state, '', stripped);
				}

				const port = createHttpTaskFormPort();
				const fetched = await port.fetch({
					stream,
					jobId: anchor.jobId,
					taskId: anchor.taskId
				});
				if (cancelled) return;

				playJob = anchor;
				bundle = fetched;
			} catch (err) {
				if (cancelled) return;
				error = err instanceof Error ? err.message : String(err);
			} finally {
				if (!cancelled) loading = false;
			}
		})();

		return () => {
			cancelled = true;
		};
	});

	function onCommitted(outcome: CommitOutcome) {
		if (!addr) return;
		const decision = decideCommitFollowThrough(outcome, addr);
		if (decision.kind === 'stay') return;
		applyCommitFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: navigateOrgTarget
		});
	}
</script>

<div class="mx-auto max-w-3xl space-y-4" data-testid="play-route" data-route-key={routeKey}>
	<header class="space-y-1" data-testid="play-chrome">
		<p class="text-lg font-semibold text-zinc-900" data-testid="play-chrome-stream">{titleText}</p>
		<a
			href={dashboardHref}
			class="inline-block text-sm text-zinc-600 underline-offset-2 hover:underline"
			data-testid="play-chrome-dashboard"
			data-sveltekit-reload
			rel="external"
			target="_self"
		>
			{m.back_to_stream_dashboard()}
		</a>
	</header>

	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
		</p>
	{/if}

	{#if loading}
		<p class="text-sm text-zinc-500" data-testid="play-load-status">{m.loading()}</p>
	{:else if error}
		<p class="text-sm text-red-700" data-testid="play-load-error" role="alert">{error}</p>
	{:else if bundle && addr}
		<TaskForm
			{bundle}
			jobId={addr.jobId}
			taskId={addr.taskId}
			jobInstanceId=""
			stream={addr.stream}
			play={true}
			onCommitted={onCommitted}
		/>
	{/if}
</div>
