<script lang="ts">
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import TaskForm from '../task-form.svelte';
	import { type TaskAddress, taskAddressKey } from '../task-address.js';
	import type { TimelineSubmission } from '../timeline/types.js';
	import { boardFormCache, type BoardFormEntry } from './board-form-cache.svelte.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';
	import BoardManyRecap from './board-many-recap.svelte';
	import BoardOnceRecap from './board-once-recap.svelte';
	import { stayConfirmSurface } from './commit-stay-recap.js';

	type Props = {
		address: TaskAddress;
		entry: BoardFormEntry | undefined;
		submitMany?: boolean;
		submissions?: TimelineSubmission[];
		focused?: boolean;
		disabled?: boolean;
		/** Stay Commit: land on once-recap / close many composer (no under-form Continue). */
		confirmRecap?: boolean;
		open?: boolean;
		/** `?seed.*` from timeline href — locks those fields for this embed open. */
		seed?: Record<string, string>;
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
		seed = {},
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
		Boolean(confirmRecap) ||
			(recapEntry?.status === 'ready' && (recapEntry.table.rows.length ?? 0) > 0)
	);
	const manyEmpty = $derived(submissions.length === 0 && !confirmRecap);

	let composerOpen = $state(false);
	/** Empty land shows the form; Add only once submissions exist. */
	const showComposer = $derived(composerOpen || manyEmpty);

	/** One-shot: stay → close composer; recap table stays visible. */
	let closedComposerOnStay = $state(false);
	$effect(() => {
		if (!confirmRecap) {
			closedComposerOnStay = false;
			return;
		}
		if (stayConfirmSurface(submitMany) !== 'many-recap') return;
		if (closedComposerOnStay) return;
		closedComposerOnStay = true;
		composerOpen = false;
		onOpenChange?.(true);
	});

	function openComposer() {
		composerOpen = true;
		onOpenChange?.(true);
		onFocus?.();
	}

	function discardComposer() {
		composerOpen = false;
		boardFormCache.beginFreshDraft(address);
	}
</script>

{#snippet discardAction()}
	<Button
		type="button"
		variant="outline"
		size="sm"
		data-testid="embed-composer-discard"
		onclick={discardComposer}
	>
		<XIcon data-icon="inline-start" aria-hidden="true" />
		{m.embed_composer_discard()}
	</Button>
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
				submitStart={submitMany && showComposer ? discardAction : undefined}
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
	data-composer={submitMany ? (showComposer ? 'open' : 'closed') : undefined}
	data-open={open ? '1' : '0'}
	data-confirm-recap={confirmRecap ? '1' : '0'}
>
	{#if open}
		<div
			id={bodyId}
			class="substep-org-body w-full min-w-0 gap-2"
			data-testid="embed-panel-body"
		>
			{#if submitMany}
				{#if !manyEmpty && !address.provisionalJob && address.jobInstanceId}
					<BoardManyRecap {address} empty={false} data-testid="embed-many-recap" />
				{/if}
				<div class={manyEmpty ? undefined : 'mt-8'} data-testid="embed-composer-region">
					{#if !showComposer}
						<div class="flex justify-end">
							<Button
								type="button"
								variant="default"
								size="sm"
								data-testid="embed-composer-open"
								onclick={openComposer}
							>
								<PlusIcon data-icon="inline-start" aria-hidden="true" />
								{m.embed_composer_add()}
							</Button>
						</div>
					{:else}
						<div class="outline-none" data-testid="embed-form-pane">
							{@render formPane()}
						</div>
					{/if}
				</div>
			{:else}
				<div class="outline-none" data-testid="embed-form-pane">
					{@render formPane()}
				</div>
			{/if}
		</div>
	{/if}
</div>
