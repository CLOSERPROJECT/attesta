<script lang="ts">
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import { untrack } from 'svelte';
	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import TaskForm from '../task-form.svelte';
	import { type TaskAddress, taskAddressKey } from '../task-address.js';
	import type { TimelineSubmission } from '../timeline/types.js';
	import { boardFormCache, type BoardFormEntry } from './board-form-cache.svelte.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';
	import BoardOnceRecap from './board-once-recap.svelte';
	import ManyRecapDoneBand from './many-recap-done-band.svelte';
	import {
		initialManyLand,
		reduceManyLand,
		showComposer,
		showDoneBand,
		stayConfirmSurface,
		type ManyLandSurface
	} from './commit-stay-recap.js';

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
		/** When false, stay land shows Done-band / once-recap only (no Add / live form). */
		orgActionable?: boolean;
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
		orgActionable = true,
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

	let manySurface = $state<ManyLandSurface>('empty-compose');

	/** Re-init when address key changes only — remount / draftGeneration / catch-up must not re-reduce. */
	$effect.pre(() => {
		if (!submitMany) return;
		void key;
		manySurface = initialManyLand({
			submissionCount: untrack(() => submissions.length),
			confirmRecap: untrack(() => confirmRecap)
		});
	});

	const composerVisible = $derived(showComposer(manySurface));
	const doneBandVisible = $derived(showDoneBand(manySurface));

	/** One-shot per confirmRecap edge — parent onOpenChange identity must not re-close Add. */
	let latchedStayConfirm = $state(false);
	$effect(() => {
		if (!confirmRecap) {
			latchedStayConfirm = false;
			return;
		}
		if (stayConfirmSurface(submitMany) !== 'many-recap') return;
		if (latchedStayConfirm) return;
		latchedStayConfirm = true;
		untrack(() => onOpenChange?.(true));
	});

	function openComposer() {
		manySurface = reduceManyLand(manySurface, { type: 'openComposer' });
		onOpenChange?.(true);
		onFocus?.();
	}

	function discardComposer() {
		manySurface = reduceManyLand(manySurface, {
			type: 'discard',
			submissionCount: submissions.length
		});
		boardFormCache.beginFreshDraft(address);
	}

	/** Every many Commit (incl. 2nd+) — surface owns close; confirmRecap is not a mode event. */
	function onFormCommitted(outcome: CommitOutcome) {
		if (submitMany) {
			manySurface = reduceManyLand(manySurface, { type: 'commitSuccess' });
		}
		onCommitted?.(outcome);
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
				submitStart={submitMany && manySurface === 'composing' ? discardAction : undefined}
				onCommitted={onFormCommitted}
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
	data-composer={submitMany ? (composerVisible ? 'open' : 'closed') : undefined}
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
				{#if doneBandVisible && !address.provisionalJob && address.jobInstanceId}
					<ManyRecapDoneBand {address} />
				{/if}
				{#if orgActionable}
					<div
						class={doneBandVisible ? 'pt-5' : undefined}
						data-testid="embed-composer-region"
					>
						{#if !composerVisible}
							<div class="flex justify-center">
								<Button
									type="button"
									variant="default"
									size="default"
									class="w-56 max-w-full shrink"
									data-testid="embed-composer-open"
									onclick={openComposer}
								>
									<PlusIcon data-icon="inline-start" aria-hidden="true" />
									{m.embed_composer_add()}
								</Button>
							</div>
						{:else}
							<div class="outline-none pt-2 pb-2" data-testid="embed-form-pane">
								<h3
									class="m-0 mb-8 flex items-center gap-1.5 border-b border-border pb-2 text-sm font-bold text-foreground"
									data-testid="embed-many-new-entry-title"
								>
									<PencilIcon class="size-3.5 shrink-0" aria-hidden="true" />
									{m.embed_many_new_entry()}
								</h3>
								{@render formPane()}
							</div>
						{/if}
					</div>
				{/if}
			{:else if !orgActionable}
				<BoardOnceRecap {address} data-testid="embed-once-recap" />
			{:else}
				<div class="outline-none" data-testid="embed-form-pane">
					{@render formPane()}
				</div>
			{/if}
		</div>
	{/if}
</div>
