<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';
	import { cn } from '#lib/shadcn/utils.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { getBoardTimelineModel } from '../board/board-timeline-model.svelte.js';
	import { boardFormCache } from '../board/board-form-cache.svelte.js';
	import BoardManyRecap from '../board/board-many-recap.svelte';
	import BoardOnceRecap from '../board/board-once-recap.svelte';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import EmbeddedTaskPanel from '../board/embedded-task-panel.svelte';
	import { parseTaskHref } from '../board/board-url.js';
	import { taskAddressKey } from '../task-address.js';
	import { streamStatusChipVisible, streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
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
	const timelineModel = getBoardTimelineModel();
	const nestCollapse = getBoardNestCollapse();

	const domId = $derived(substep.substepId.replaceAll('.', '-'));
	const headingId = $derived(`substep-${domId}-heading`);
	const embedBodyId = $derived(`substep-${domId}-embed`);
	const status = $derived(substep.status);
	const statusChipLabel = $derived(status ? streamStatusLabel(status) : '');
	const showStatusChip = $derived(!!status && streamStatusChipVisible(status, timelineModel.showAll));
	const submissions = $derived(substep.submissions ?? []);
	const statusClass = $derived(status ? `substep-${status}` : '');

	const parsed = $derived(
		substep.href ? parseTaskHref(substep.href, stream, instance) : null
	);
	const address = $derived(parsed?.address ?? null);
	const seed = $derived(parsed?.seed ?? {});
	const embed = $derived(!!(substep.orgActionable && address));
	const focused = $derived.by(() => {
		if (!address) return false;
		return timelineModel.isSelected(address);
	});
	const entry = $derived.by(() => {
		void boardFormCache.entries;
		if (!address) return undefined;
		return boardFormCache.get(taskAddressKey(address));
	});
	const confirmRecap = $derived(address ? board.confirmRecapFor(address) : false);
	const embedOpen = $derived(address ? timelineModel.isBodyOpen(address) : false);

	let detailsOpen = $state(false);
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		detailsOpen = false;
	});

	function toggleEmbed() {
		if (!address) return;
		const next = timelineModel.toggleBody(address);
		// Closing must not select — select() always re-opens the body.
		if (next) timelineModel.select(address);
	}

	/** Padding/gaps on the summary toggle; ignore Form|Recap tablist. */
	function onEmbedHeaderClick(e: MouseEvent) {
		const t = e.target;
		if (!(t instanceof Element)) return;
		if (t.closest('[data-testid="embed-form-recap-tabs"], [role="tablist"]')) return;
		toggleEmbed();
	}

	function onEmbedToggleClick(e: MouseEvent) {
		e.stopPropagation();
		toggleEmbed();
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
			<EmbeddedTaskPanel
				{address}
				{entry}
				{seed}
				submitMany={!!substep.submitMany}
				{submissions}
				{focused}
				{confirmRecap}
				open={embedOpen}
				bodyId={embedBodyId}
				onOpenChange={(next) => timelineModel.setBodyOpen(address, next)}
				onFocus={() => timelineModel.select(address)}
				onCommitted={(outcome) =>
					board.onCommitted(address, outcome, { submitMany: !!substep.submitMany })
				}
			>
				{#snippet header({ tablist })}
					<!--
						Header is a div: title button · optional Form|Recap tablist · chevron button.
						Never nest tablist inside the expand trigger (interactive-in-button a11y).
						Div click covers padding/gaps (CSS cursor:pointer); tablist clicks are ignored.
						Title/chevron buttons keep keyboard a11y and stopPropagation to avoid double-toggle.
					-->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class={cn(
							'group/accordion-trigger substep-accordion-summary substep-card-link relative flex w-full flex-1 items-center justify-between text-sm font-medium',
							'**:data-[slot=accordion-trigger-icon]:text-muted-foreground **:data-[slot=accordion-trigger-icon]:size-4'
						)}
						data-state={embedOpen ? 'open' : 'closed'}
						onclick={onEmbedHeaderClick}
					>
						<button
							type="button"
							class="substep-embed-title-trigger min-w-0 flex-1 border-0 bg-transparent p-0 text-left outline-none hover:no-underline"
							aria-expanded={embedOpen}
							aria-controls={embedBodyId}
							data-slot="collapsible-trigger"
							data-state={embedOpen ? 'open' : 'closed'}
							onclick={onEmbedToggleClick}
						>
							{#if showStatusChip}
								<TimelineHeaderLine {mark} title={substep.title} titleId={headingId}>
									{#snippet trail()}
										<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
									{/snippet}
								</TimelineHeaderLine>
							{:else}
								<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} />
							{/if}
						</button>
						{#if tablist}
							{@render tablist()}
						{/if}
						<button
							type="button"
							class="substep-embed-chevron-trigger border-0 bg-transparent outline-none"
							aria-expanded={embedOpen}
							aria-controls={embedBodyId}
							aria-labelledby={headingId}
							data-slot="collapsible-trigger"
							data-state={embedOpen ? 'open' : 'closed'}
							onclick={onEmbedToggleClick}
						>
							<ChevronDownIcon
								data-slot="accordion-trigger-icon"
								class={cn('pointer-events-none shrink-0', embedOpen && 'hidden')}
							/>
							<ChevronUpIcon
								data-slot="accordion-trigger-icon"
								class={cn('pointer-events-none shrink-0', !embedOpen && 'hidden')}
							/>
						</button>
					</div>
				{/snippet}
			</EmbeddedTaskPanel>
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
				{#if showStatusChip}
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
					{#if substep.blockedReason || (submissions.length > 0 && address)}
						<div class="substep-org-body">
							{#if substep.blockedReason}
								<p
									class="m-0 whitespace-pre-line text-secondary-foreground"
									data-testid="task-blocked-reason"
								>
									{substep.blockedReason}
								</p>
							{/if}
							{#if submissions.length > 0 && address}
								{#if substep.submitMany}
									<BoardManyRecap {address} data-testid="timeline-many-recap" />
								{:else}
									<BoardOnceRecap {address} data-testid="timeline-once-recap" />
								{/if}
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
