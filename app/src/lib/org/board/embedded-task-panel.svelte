<script lang="ts">
	import ListIcon from '@lucide/svelte/icons/list';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import type { Snippet } from 'svelte';
	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as Tabs from '#lib/shadcn/ui/tabs/index.js';
	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import TaskForm from '../task-form.svelte';
	import { taskAddressKey, type TaskAddress } from '../task-address.js';
	import type { TimelineSubmission } from '../timeline/types.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';
	import BoardManyRecap from './board-many-recap.svelte';
	import BoardOnceRecap from './board-once-recap.svelte';
	import type { BoardFormEntry } from './board-form-cache.svelte.js';
	import { stayConfirmSurface } from './commit-stay-recap.js';

	type Tab = 'form' | 'recap';

	type HeaderArgs = {
		/** New|Done tablist when submitMany and open; else undefined. */
		tablist: Snippet | undefined;
	};

	type Props = {
		address: TaskAddress;
		entry: BoardFormEntry | undefined;
		submitMany?: boolean;
		submissions?: TimelineSubmission[];
		focused?: boolean;
		disabled?: boolean;
		/** Stay Commit: land on once-recap / Recap tab (no under-form Continue). */
		confirmRecap?: boolean;
		open?: boolean;
		tab?: Tab;
		/** `?seed.*` from timeline href — locks those fields for this embed open. */
		seed?: Record<string, string>;
		/** Substep header row (title · tablist · chevron). Keeps tablist under Tabs.Root. */
		header?: Snippet<[HeaderArgs]>;
		/** Optional id for the embed body region (aria-controls). */
		bodyId?: string;
		onOpenChange?: (open: boolean) => void;
		onCommitted?: (outcome: CommitOutcome) => void;
		onFocus?: () => void;
		onAcceptStale?: () => void;
	};

	let {
		address,
		entry,
		submitMany = false,
		submissions = [],
		focused = false,
		disabled = false,
		confirmRecap = false,
		open = true,
		tab = $bindable<Tab>('form'),
		seed = {},
		header,
		bodyId,
		onOpenChange,
		onCommitted,
		onFocus,
		onAcceptStale
	}: Props = $props();

	const status = $derived(entry?.status ?? 'idle');
	const bundle = $derived(entry?.bundle ?? null);
	const error = $derived(entry?.error ?? null);
	const key = $derived(taskAddressKey(address));
	const recapEntry = $derived.by(() => {
		void boardRecapCache.entries;
		return boardRecapCache.get(key);
	});
	const showOnceRecap = $derived(
		Boolean(confirmRecap) || (recapEntry?.status === 'ready' && (recapEntry.items.length ?? 0) > 0)
	);
	const manyEmpty = $derived(submissions.length === 0 && !confirmRecap);

	/** One-shot: stay → Recap tab once; user may switch back to Form. */
	let landedManyRecap = $state(false);
	$effect(() => {
		if (!confirmRecap) {
			landedManyRecap = false;
			return;
		}
		if (stayConfirmSurface(submitMany) !== 'many-recap-tab') return;
		if (landedManyRecap) return;
		landedManyRecap = true;
		tab = 'recap';
		onOpenChange?.(true);
	});

	function selectTab(next: Tab) {
		tab = next;
		onOpenChange?.(true);
		if (next === 'form') onFocus?.();
	}
</script>

{#snippet formRecapTablist()}
	<Tabs.List variant="line" class="embed-tabs-list p-0" data-testid="embed-form-recap-tabs">
		<Tabs.Trigger
			value="form"
			class="embed-tabs-trigger"
			data-testid="embed-tab-form"
			onclick={() => selectTab('form')}
		>
			<PlusIcon data-icon="inline-start" aria-hidden="true" />
			{m.embed_tab_form()}
		</Tabs.Trigger>
		<Tabs.Trigger
			value="recap"
			class="embed-tabs-trigger"
			data-testid="embed-tab-recap"
			onclick={() => selectTab('recap')}
		>
			<ListIcon data-icon="inline-start" aria-hidden="true" />
			{m.embed_tab_recap()}
		</Tabs.Trigger>
	</Tabs.List>
{/snippet}

{#snippet formPane()}
	{#if !submitMany && confirmRecap}
		<BoardOnceRecap {address} data-testid="embed-once-recap" />
	{:else if status === 'loading' || status === 'idle'}
		<p class="m-0 text-secondary-foreground" data-testid="embed-form-skeleton">{m.loading()}</p>
	{:else if status === 'error'}
		<p class="text-sm text-red-700" data-testid="embed-form-error" role="alert">
			{error}
		</p>
	{:else if status === 'ready' && bundle}
		{#if entry?.stale}
			<div
				class="flex items-center justify-between gap-2 rounded-md border border-border bg-muted/60 px-3 py-2 text-sm"
				role="status"
				data-testid="embed-stale-notice"
			>
				<span>{m.embed_stale_notice()}</span>
				<Button
					type="button"
					variant="outline"
					size="sm"
					data-testid="embed-stale-reload"
					onclick={() => onAcceptStale?.()}
				>
					{m.embed_stale_reload()}
				</Button>
			</div>
		{/if}
		{#key `${entry?.draftGeneration ?? 0}|${Object.keys(seed).sort().join(',')}`}
			<TaskForm
				stream={address.stream}
				instance={address.instance}
				jobId={address.jobId}
				taskId={address.taskId}
				jobInstanceId={address.jobInstanceId}
				{bundle}
				provisionalJob={address.provisionalJob}
				{seed}
				{disabled}
				onCommitted={(outcome) => onCommitted?.(outcome)}
			/>
		{/key}
		{#if !submitMany && showOnceRecap}
			<BoardOnceRecap {address} data-testid="embed-once-recap" />
		{/if}
	{/if}
{/snippet}

<div
	class="substep-embed"
	data-testid="embedded-task-panel"
	data-address-key={key}
	data-focused={focused ? '1' : '0'}
	data-tab={tab}
	data-open={open ? '1' : '0'}
	data-confirm-recap={confirmRecap ? '1' : '0'}
>
	{#if submitMany}
		<Tabs.Root bind:value={tab} class="w-full min-w-0 flex-col gap-0">
			{#if header}
				{@render header({ tablist: open ? formRecapTablist : undefined })}
			{:else if open}
				{@render formRecapTablist()}
			{/if}

			{#if open}
				<div
					id={bodyId}
					class="substep-org-body w-full min-w-0 gap-2"
					data-testid="embed-panel-body"
				>
					<Tabs.Content value="form" class="outline-none" data-testid="embed-form-pane">
						{@render formPane()}
					</Tabs.Content>
					<Tabs.Content value="recap" class="outline-none" data-testid="embed-recap-pane">
						{#if !address.provisionalJob && address.jobInstanceId}
							<BoardManyRecap {address} empty={manyEmpty} data-testid="embed-many-recap" />
						{/if}
					</Tabs.Content>
				</div>
			{/if}
		</Tabs.Root>
	{:else}
		{#if header}
			{@render header({ tablist: undefined })}
		{/if}

		{#if open}
			<div id={bodyId} class="substep-org-body w-full min-w-0 gap-2" data-testid="embed-panel-body">
				<div class="outline-none" data-testid="embed-form-pane">
					{@render formPane()}
				</div>
			</div>
		{/if}
	{/if}
</div>
