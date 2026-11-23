<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';
	import { cn } from '#lib/shadcn/utils.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { boardFormCache } from '../board/board-form-cache.svelte.js';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import EmbeddedTaskPanel from '../board/embedded-task-panel.svelte';
	import { parseTaskHref } from '../board/board-url.js';
	import { taskAddressKey } from '../task-address.js';
	import { streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineSubmissions from './timeline-submissions.svelte';
	import type { TimelineSubstep } from './types.js';

	type Props = {
		substep: TimelineSubstep;
		/** Hierarchical outline mark, e.g. 2A.1 or 2.1. */
		mark: string;
		stream?: string;
		instance?: string;
	};

	let { substep, mark, stream = '', instance = '' }: Props = $props();

	const board = getBoardEmbed();
	const nestCollapse = getBoardNestCollapse();

	const domId = $derived(substep.substepId.replaceAll('.', '-'));
	const headingId = $derived(`substep-${domId}-heading`);
	const status = $derived(substep.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');
	const submissions = $derived(substep.submissions ?? []);
	const statusClass = $derived(status ? `substep-${status}` : '');

	const parsed = $derived(
		substep.href ? parseTaskHref(substep.href, stream, instance) : null
	);
	const address = $derived(parsed?.address ?? null);
	const embed = $derived(!!(substep.orgActionable && address));
	const focused = $derived.by(() => {
		if (!address) return false;
		return board.isFocused(address);
	});
	const entry = $derived.by(() => {
		void boardFormCache.entries;
		if (!address) return undefined;
		return boardFormCache.get(taskAddressKey(address));
	});
	const stayedOutcome = $derived(address ? board.stayedFor(address) : null);
	const embedOpen = $derived(address ? board.isEmbedOpen(address) : false);

	let detailsOpen = $state(false);
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		detailsOpen = false;
	});

	function toggleEmbed() {
		if (!address) return;
		const next = board.toggleEmbedOpen(address);
		if (next) board.setFocus(address);
	}
</script>

<li
	class={['substep', statusClass]}
	data-org-actionable={substep.orgActionable ? 'true' : undefined}
	data-board-focused={focused ? 'true' : undefined}
>
	{#if embed && address}
		<div
			class="substep-accordion js-process-substep-panel substep-embed-host"
			data-substep-id={substep.substepId}
			data-testid="actionable-embed"
			data-state={embedOpen ? 'open' : 'closed'}
			data-slot="collapsible"
			aria-labelledby={headingId}
		>
			<button
				type="button"
				class={cn(
					'group/accordion-trigger substep-accordion-summary substep-card-link relative flex w-full flex-1 items-center justify-between border-0 bg-transparent text-left text-sm font-medium outline-none hover:no-underline',
					'**:data-[slot=accordion-trigger-icon]:text-muted-foreground **:data-[slot=accordion-trigger-icon]:size-4'
				)}
				aria-expanded={embedOpen}
				aria-controls={`substep-${domId}-embed`}
				data-slot="collapsible-trigger"
				data-state={embedOpen ? 'open' : 'closed'}
				onclick={toggleEmbed}
			>
				{#if status}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId}>
						{#snippet trail()}
							<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
						{/snippet}
					</TimelineHeaderLine>
				{:else}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} />
				{/if}
				<ChevronDownIcon
					data-slot="accordion-trigger-icon"
					class={cn('pointer-events-none shrink-0', embedOpen && 'hidden')}
				/>
				<ChevronUpIcon
					data-slot="accordion-trigger-icon"
					class={cn('pointer-events-none shrink-0', !embedOpen && 'hidden')}
				/>
			</button>
			<div id={`substep-${domId}-embed`}>
				<EmbeddedTaskPanel
					{address}
					{entry}
					submitMany={!!substep.submitMany}
					{submissions}
					{focused}
					play={!!board.play}
					stayedOutcome={stayedOutcome}
					open={embedOpen}
					onOpenChange={(next) => board.setEmbedOpen(address, next)}
					onFocus={() => board.setFocus(address)}
					onCommitted={(outcome) => board.onCommitted(address, outcome)}
					onContinue={() => board.onContinue(address)}
				/>
			</div>
		</div>
	{:else}
		<Collapsible.Root
			class="substep-accordion js-process-substep-panel"
			data-substep-id={substep.substepId}
			data-testid={focused ? 'focused-substep' : undefined}
			aria-labelledby={headingId}
			bind:open={detailsOpen}
		>
			<Collapsible.Trigger class="substep-accordion-summary w-full items-center hover:no-underline">
				{#if status}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId}>
						{#snippet trail()}
							<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
						{/snippet}
					</TimelineHeaderLine>
				{:else}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} />
				{/if}
			</Collapsible.Trigger>
			<Collapsible.Content>
				<div class="substep-details">
					{#if substep.blockedReason || submissions.length > 0}
						<div class="substep-org-body">
							{#if substep.blockedReason}
								<p
									class="m-0 whitespace-pre-line text-secondary-foreground"
									data-testid="task-blocked-reason"
								>
									{substep.blockedReason}
								</p>
							{/if}
							{#if submissions.length > 0}
								<TimelineSubmissions {submissions} open={substep.status === 'done'} />
							{/if}
						</div>
					{:else}
						<p class="text-secondary-foreground">No data form configured for this substep</p>
					{/if}
				</div>
			</Collapsible.Content>
		</Collapsible.Root>
	{/if}
</li>
