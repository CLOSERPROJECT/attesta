<script lang="ts">
	import ClipboardListIcon from '@lucide/svelte/icons/clipboard-list';
	import HistoryIcon from '@lucide/svelte/icons/history';
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

	type Tab = 'form' | 'recap';

	type Props = {
		address: TaskAddress;
		entry: BoardFormEntry | undefined;
		submitMany?: boolean;
		submissions?: TimelineSubmission[];
		focused?: boolean;
		play?: boolean;
		disabled?: boolean;
		stayedOutcome?: CommitOutcome | null;
		open?: boolean;
		tab?: Tab;
		onOpenChange?: (open: boolean) => void;
		onCommitted?: (outcome: CommitOutcome) => void;
		onContinue?: () => void;
		onFocus?: () => void;
	};

	let {
		address,
		entry,
		submitMany = false,
		submissions = [],
		focused = false,
		play = false,
		disabled = false,
		stayedOutcome = null,
		open = true,
		tab = $bindable<Tab>('form'),
		onOpenChange,
		onCommitted,
		onContinue,
		onFocus
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
		Boolean(stayedOutcome) ||
			(recapEntry?.status === 'ready' && (recapEntry.items.length ?? 0) > 0)
	);
	const manyEmpty = $derived(submissions.length === 0 && !stayedOutcome);

	function selectTab(next: Tab) {
		tab = next;
		onOpenChange?.(true);
		if (next === 'form') onFocus?.();
	}
</script>

<div
	class="substep-embed"
	data-testid="embedded-task-panel"
	data-address-key={key}
	data-focused={focused ? '1' : '0'}
	data-tab={tab}
	data-open={open ? '1' : '0'}
>
	<Tabs.Root bind:value={tab} class="w-full min-w-0 flex-col gap-0">
		{#if open}
			<Tabs.List
				variant="line"
				class="embed-tabs-list"
				data-testid={submitMany ? 'embed-form-recap-tabs' : undefined}
				aria-label={submitMany ? undefined : m.embed_tab_form()}
			>
				<Tabs.Trigger
					value="form"
					class="embed-tabs-trigger"
					data-testid="embed-tab-form"
					onclick={() => selectTab('form')}
				>
					<ClipboardListIcon data-icon="inline-start" aria-hidden="true" />
					{m.embed_tab_form()}
				</Tabs.Trigger>
				{#if submitMany}
					<Tabs.Trigger
						value="recap"
						class="embed-tabs-trigger"
						data-testid="embed-tab-recap"
						onclick={() => selectTab('recap')}
					>
						<HistoryIcon data-icon="inline-start" aria-hidden="true" />
						{m.embed_tab_recap()}
					</Tabs.Trigger>
				{/if}
			</Tabs.List>

			<div class="substep-org-body w-full min-w-0 gap-2" data-testid="embed-panel-body">
				<Tabs.Content value="form" class="outline-none" data-testid="embed-form-pane">
					{#if status === 'loading' || status === 'idle'}
						<p class="m-0 text-secondary-foreground" data-testid="embed-form-skeleton">{m.loading()}</p>
					{:else if status === 'error'}
						<p class="text-sm text-red-700" data-testid="embed-form-error" role="alert">
							{error}
						</p>
					{:else if status === 'ready' && bundle}
						<TaskForm
							stream={address.stream}
							instance={address.instance}
							jobId={address.jobId}
							taskId={address.taskId}
							jobInstanceId={address.jobInstanceId}
							{bundle}
							provisionalJob={address.provisionalJob}
							{play}
							disabled={disabled || Boolean(stayedOutcome)}
							onCommitted={(outcome) => onCommitted?.(outcome)}
						/>
						{#if stayedOutcome}
							<div class="space-y-3" data-testid="task-links-stay">
								<Button
									type="button"
									variant="outline"
									size="sm"
									class="bg-[var(--input)]"
									data-testid="task-links-continue"
									onclick={() => onContinue?.()}
								>
									{m.continue()}
								</Button>
							</div>
						{/if}
						{#if !submitMany && showOnceRecap}
							<BoardOnceRecap {address} data-testid="embed-once-recap" />
						{/if}
					{/if}
				</Tabs.Content>

				{#if submitMany}
					<Tabs.Content value="recap" class="outline-none" data-testid="embed-recap-pane">
						{#if !address.provisionalJob && address.jobInstanceId}
							<BoardManyRecap
								{address}
								empty={manyEmpty}
								data-testid="embed-many-recap"
							/>
						{/if}
					</Tabs.Content>
				{/if}
			</div>
		{/if}
	</Tabs.Root>
</div>
