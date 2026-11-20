<script lang="ts">
	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as Tabs from '#lib/shadcn/ui/tabs/index.js';
	import { cn } from '#lib/shadcn/utils.js';
	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import ManySubmitHistory from '../many-submit-history.svelte';
	import TaskForm from '../task-form.svelte';
	import { taskAddressKey, type TaskAddress } from '../task-address.js';
	import type { TimelineSubmission } from '../timeline/types.js';
	import TimelineSubmissions from '../timeline/timeline-submissions.svelte';
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
	const recapRequest = $derived({
		stream: address.stream,
		instance: address.instance || undefined,
		jobId: address.jobId,
		taskId: address.taskId,
		jobInstanceId: address.provisionalJob ? 'new' : address.jobInstanceId
	});

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
				class={cn(
					'flex h-auto w-full min-w-0 shrink-0 rounded-none border-b border-border p-0'
				)}
				data-testid={submitMany ? 'embed-form-recap-tabs' : undefined}
				aria-label={submitMany ? undefined : m.embed_tab_form()}
			>
				<Tabs.Trigger
					value="form"
					class="h-auto min-h-9 flex-1 justify-center rounded-none px-0 py-2 text-center shadow-none hover:bg-black/5 hover:text-foreground"
					data-testid="embed-tab-form"
					onclick={() => selectTab('form')}
				>
					{m.embed_tab_form()}
				</Tabs.Trigger>
				{#if submitMany}
					<Tabs.Trigger
						value="recap"
						class="h-auto min-h-9 flex-1 justify-center rounded-none px-0 py-2 text-center shadow-none hover:bg-black/5 hover:text-foreground"
						data-testid="embed-tab-recap"
						onclick={() => selectTab('recap')}
					>
						{m.embed_tab_recap()}
					</Tabs.Trigger>
				{/if}
			</Tabs.List>

			<div class="substep-org-body w-full min-w-0 gap-2" data-testid="embed-panel-body">
				<Tabs.Content value="form" class="outline-none" data-testid="embed-form-pane">
					{#if status === 'loading' || status === 'idle'}
						<p class="muted u-m-0" data-testid="embed-form-skeleton">{m.loading()}</p>
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
					{/if}
				</Tabs.Content>

				{#if submitMany}
					<Tabs.Content value="recap" class="outline-none" data-testid="embed-recap-pane">
						{#if submissions.length > 0}
							<TimelineSubmissions {submissions} open={true} />
						{/if}
						{#if !address.provisionalJob && address.jobInstanceId}
							<ManySubmitHistory
								submit="many"
								submissionCount={submissions.length}
								request={recapRequest}
								data-testid="embed-many-submit-history"
							/>
						{/if}
					</Tabs.Content>
				{/if}
			</div>
		{/if}
	</Tabs.Root>
</div>
