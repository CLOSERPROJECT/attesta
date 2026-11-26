<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { getBoardTimelineModel } from '../board/board-timeline-model.svelte.js';
	import { boardFormCache } from '../board/board-form-cache.svelte.js';
	import { boardJustCompleted } from '../board/board-just-completed.svelte.js';
	import { getBoardNestCollapse } from '../board/board-nest-collapse.svelte.js';
	import EmbeddedTaskPanel from '../board/embedded-task-panel.svelte';
	import { parseTaskHref } from '../board/board-url.js';
	import DraftBozzaMark from '../draft-bozza-mark.svelte';
	import { draftAddressForTask } from '../drafts.js';
	import { orgSession } from '../org-app.svelte.js';
	import { taskAddressKey } from '../task-address.js';
	import { streamStatusChipVisible, streamStatusLabel } from './stream-status-label.js';
	import TimelineHeaderLine from './timeline-header-line.svelte';
	import TimelineRoleBadges from './timeline-role-badges.svelte';
	import { timelineRoleBadgesVisible } from './timeline-role-badge-label.js';
	import { timelineSubstepShowsTaskPanel } from './timeline-substep-embed.js';
	import { timelineSubstepTrailVisible } from './timeline-substep-trail.js';
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
	const showStatusChip = $derived(
		!!status && streamStatusChipVisible(status, timelineModel.showAll)
	);
	const roleBadges = $derived(
		timelineRoleBadgesVisible(timelineModel.showAll, substep.roleBadges)
	);
	const submissions = $derived(substep.submissions ?? []);
	const statusClass = $derived(status ? `substep-${status}` : '');

	const parsed = $derived(substep.href ? parseTaskHref(substep.href, stream, instance) : null);
	const address = $derived(parsed?.address ?? null);
	const seed = $derived(parsed?.seed ?? {});
	const showDraft = $derived.by(() => {
		void orgSession.drafts;
		if (!address) return false;
		const d = draftAddressForTask({
			jobId: address.jobId,
			taskId: address.taskId,
			jobInstanceId: address.jobInstanceId,
			seed: Object.keys(seed).length > 0 ? seed : undefined
		});
		return orgSession.hasDirtyDraft(d.taskPath, d.draftBagInstanceKey);
	});
	const showTrail = $derived(
		timelineSubstepTrailVisible({
			showStatusChip,
			roleBadgeCount: roleBadges.length,
			showDraft
		})
	);
	const focused = $derived.by(() => {
		if (!address) return false;
		return timelineModel.isSelected(address, seed);
	});
	const entry = $derived.by(() => {
		void boardFormCache.entries;
		if (!address) return undefined;
		return boardFormCache.get(taskAddressKey(address));
	});
	const confirmRecap = $derived(address ? board.confirmRecapFor(address) : false);
	const justCompleted = $derived(address ? boardJustCompleted.has(address) : false);
	const showTaskPanel = $derived(
		timelineSubstepShowsTaskPanel({
			hasAddress: !!address,
			orgActionable: !!substep.orgActionable,
			confirmRecap,
			justCompleted,
			submissionCount: submissions.length
		})
	);
	const bodyOpen = $derived(address ? timelineModel.isBodyOpen(address, seed) : false);

	let detailsOpen = $state(false);
	$effect(() => {
		const e = nestCollapse.epoch;
		if (e === 0) return;
		detailsOpen = false;
	});

	function onBodyOpenChange(next: boolean) {
		if (!address) return;
		timelineModel.setBodyOpen(address, next, seed);
		// Closing must not select — select() always re-opens the body.
		if (next) timelineModel.select(address, seed);
	}
</script>

{#snippet trail()}
	{#if roleBadges.length > 0}
		<TimelineRoleBadges badges={roleBadges} />
	{/if}
	{#if showDraft}
		<DraftBozzaMark />
	{/if}
	{#if showStatusChip}
		<span class="status" data-status={status} aria-label={statusChipLabel}>{statusChipLabel}</span>
	{/if}
{/snippet}

<li
	class={['substep', statusClass]}
	data-org-actionable={substep.orgActionable ? 'true' : undefined}
	data-board-focused={focused ? 'true' : undefined}
>
	{#if address}
		<Collapsible.Root
			class="substep-accordion js-process-substep-panel substep-embed-host"
			data-substep-id={substep.substepId}
			data-testid="actionable-embed"
			data-state={bodyOpen ? 'open' : 'closed'}
			aria-labelledby={headingId}
			open={bodyOpen}
			onOpenChange={onBodyOpenChange}
		>
			<Collapsible.Trigger class="substep-accordion-summary substep-card-link w-full items-center hover:no-underline">
				{#if showTrail}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} {trail} />
				{:else}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} />
				{/if}
			</Collapsible.Trigger>
			<Collapsible.Content>
				{#if showTaskPanel}
					<EmbeddedTaskPanel
						{address}
						{entry}
						{seed}
						submitMany={!!substep.submitMany}
						{submissions}
						{focused}
						{confirmRecap}
						orgActionable={!!substep.orgActionable}
						open={bodyOpen}
						bodyId={embedBodyId}
						onOpenChange={(next) => timelineModel.setBodyOpen(address, next, seed)}
						onFocus={() => timelineModel.select(address, seed)}
						onCommitted={(outcome) =>
							board.onCommitted(address, outcome, { submitMany: !!substep.submitMany })
						}
						onAcceptStale={() => board.acceptStale(address)}
					/>
				{:else}
					<div class="substep-details">
						{#if substep.blockedReason}
							<p
								class="m-0 whitespace-pre-line text-secondary-foreground"
								data-testid="task-blocked-reason"
							>
								{substep.blockedReason}
							</p>
						{:else}
							<p class="text-secondary-foreground">No data form configured for this substep</p>
						{/if}
					</div>
				{/if}
			</Collapsible.Content>
		</Collapsible.Root>
	{:else}
		<Collapsible.Root
			class="substep-accordion js-process-substep-panel"
			data-substep-id={substep.substepId}
			data-testid={focused ? 'focused-substep' : undefined}
			aria-labelledby={headingId}
			bind:open={detailsOpen}
		>
			<Collapsible.Trigger class="substep-accordion-summary w-full items-center hover:no-underline">
				{#if showTrail}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} {trail} />
				{:else}
					<TimelineHeaderLine {mark} title={substep.title} titleId={headingId} />
				{/if}
			</Collapsible.Trigger>
			<Collapsible.Content>
				<div class="substep-details">
					{#if substep.blockedReason}
						<p
							class="m-0 whitespace-pre-line text-secondary-foreground"
							data-testid="task-blocked-reason"
						>
							{substep.blockedReason}
						</p>
					{:else}
						<p class="text-secondary-foreground">No data form configured for this substep</p>
					{/if}
				</div>
			</Collapsible.Content>
		</Collapsible.Root>
	{/if}
</li>