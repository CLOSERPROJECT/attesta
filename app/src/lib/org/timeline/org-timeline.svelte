<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';

	import { fetchStreamTitle } from '../stream-meta.js';
	import { createHttpTimelinePort } from './http-timeline-port.js';
	import TimelineStep from './timeline-step.svelte';
	import type { TimelineView } from './types.js';

	import './org-timeline.css';

	type Props = {
		stream: string;
		instance: string;
	};

	let { stream, instance }: Props = $props();

	const port = createHttpTimelinePort();

	let view = $state.raw<TimelineView | null>(null);
	let error = $state('');
	let loading = $state(true);
	let streamTitle = $state('');
	let showAll = $state(false);

	const hasActionable = $derived(
		!!view?.steps.some(
			(step) =>
				(step.substeps ?? []).some((s) => s.orgActionable) ||
				(step.rows ?? []).some((row) => row.substeps.some((s) => s.orgActionable))
		)
	);
	const actionableOnlyEmpty = $derived(!showAll && !hasActionable);
	const filterRootClass = $derived(
		[
			'js-show-all-filter-root',
			'stream-show-all-filter-root',
			!showAll ? 'is-actionable-only' : '',
			actionableOnlyEmpty ? 'is-actionable-only-empty' : ''
		]
			.filter(Boolean)
			.join(' ')
	);

	onMount(() => {
		void fetchStreamTitle(stream).then((title) => {
			streamTitle = title;
		});
	});

	$effect(() => {
		const s = stream;
		const i = instance;
		let cancelled = false;
		loading = true;
		error = '';
		void port
			.load({ stream: s, instance: i })
			.then((next) => {
				if (!cancelled) view = next;
			})
			.catch((e: unknown) => {
				if (cancelled) return;
				view = null;
				error =
					e instanceof Error
						? e.message
						: m.org_timeline_load_failed({ status: 0 });
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});
</script>

<div
	class="org-timeline-page process-page"
	data-testid="org-timeline"
	data-show-all={showAll ? '1' : '0'}
>
	<header class="org-timeline-header" data-testid="instance-chrome">
		<p class="org-timeline-header-stream" data-testid="instance-chrome-stream">
			{streamTitle || stream}
		</p>
		<p class="org-timeline-header-label" data-testid="instance-chrome-label">
			{view?.label || instance}
		</p>
	</header>

	{#if loading && !view}
		<p class="text-sm text-zinc-500" data-testid="org-timeline-status">{m.loading()}</p>
	{:else if error && !view}
		<p class="text-sm text-red-700" data-testid="org-timeline-error" role="alert">{error}</p>
	{:else if view}
		<div class="process-stream-block">
			<div class="stream-show-all-filter">
				<label class="stream-show-all-filter-label">
					<input
						type="checkbox"
						class="js-show-all-filter"
						checked={showAll}
						onchange={(e) => {
							showAll = e.currentTarget.checked;
						}}
					/>
					<span>Show all</span>
				</label>
			</div>
			<div class={filterRootClass}>
				<div class="empty-state stream-show-all-filter-empty" hidden={!actionableOnlyEmpty}>
					<p class="empty-state-title">No matching tasks</p>
					<p class="empty-state-hint">Turn on Show all to see the full stream timeline.</p>
				</div>
				<div class="stream-timeline-list">
					{#each view.steps as step (step.stepId)}
						<TimelineStep {step} />
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>
