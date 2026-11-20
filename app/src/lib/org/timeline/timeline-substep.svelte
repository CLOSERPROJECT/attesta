<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import { getBoardEmbed } from '../board/board-context.js';
	import { boardFormCache } from '../board/board-form-cache.svelte.js';
	import EmbeddedTaskPanel from '../board/embedded-task-panel.svelte';
	import { parseTaskHref } from '../board/board-url.js';
	import { taskAddressKey } from '../task-address.js';
	import TimelineSubmissions from './timeline-submissions.svelte';
	import type { TimelineSubstep } from './types.js';

	type Props = {
		substep: TimelineSubstep;
		stream?: string;
		instance?: string;
	};

	let { substep, stream = '', instance = '' }: Props = $props();

	const board = getBoardEmbed();

	const domId = $derived(substep.substepId.replaceAll('.', '-'));
	const statusLabel = $derived(substep.statusLabel || substep.status);
	const submissions = $derived(substep.submissions ?? []);
	const statusClass = $derived(substep.status ? `substep-${substep.status}` : '');

	const parsed = $derived(
		substep.orgActionable && substep.href
			? parseTaskHref(substep.href, stream, instance)
			: null
	);
	const address = $derived(parsed?.address ?? null);
	const embed = $derived(!!address);
	const focusHint = $derived.by(() => {
		const f = board.focus;
		if (!f) return false;
		if (address) return board.isFocused(address);
		const id = substep.substepId;
		return id === `${f.jobId}.${f.taskId}`;
	});
	const focused = $derived(focusHint);
	const entry = $derived.by(() => {
		void boardFormCache.entries;
		if (!address) return undefined;
		return boardFormCache.get(taskAddressKey(address));
	});
	const stayedOutcome = $derived(address ? board.stayedFor(address) : null);

	let embedOpen = $state(true);
	let detailsOpen = $state(false);
	let wasFocused = $state(false);

	// Rising edge only — `if (focused) open=true` re-fires when focus identity churns and fights close.
	$effect.pre(() => {
		if (focused && !wasFocused) {
			embedOpen = true;
			detailsOpen = true;
		}
		wasFocused = focused;
	});

	function onEmbedOpenChange(next: boolean) {
		embedOpen = next;
		if (next && address) board.setFocus(address);
	}
</script>

<li
	class={['substep', statusClass].filter(Boolean).join(' ')}
	data-org-actionable={substep.orgActionable ? 'true' : undefined}
	data-board-focused={focused ? 'true' : undefined}
>
	{#if embed && address}
		<Collapsible.Root
			class="substep-accordion js-process-substep-panel substep-embed-host"
			data-substep-id={substep.substepId}
			data-testid="actionable-embed"
			aria-labelledby={`substep-${domId}-heading`}
			open={embedOpen}
			onOpenChange={onEmbedOpenChange}
		>
			<Collapsible.Trigger
				class="substep-accordion-summary substep-card-link w-full items-center border-0 bg-transparent text-left hover:no-underline"
				aria-controls={`substep-${domId}-embed`}
			>
				<div class="substep-summary-main">
					<span class="pill pill-panel">{substep.substepId}</span>
					<div class="substep-title">
						<span class="substep-title-heading" id={`substep-${domId}-heading`}>
							{substep.title}
						</span>
					</div>
				</div>
				<span class="status" aria-label={`Status: ${statusLabel}`}>{statusLabel}</span>
			</Collapsible.Trigger>
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
					onOpenChange={onEmbedOpenChange}
					onFocus={() => board.setFocus(address)}
					onCommitted={(outcome) => board.onCommitted(address, outcome)}
					onContinue={() => board.onContinue(address)}
				/>
			</div>
		</Collapsible.Root>
	{:else}
		<Collapsible.Root
			class="substep-accordion js-process-substep-panel"
			data-substep-id={substep.substepId}
			data-testid={focused ? 'focused-substep' : undefined}
			aria-labelledby={`substep-${domId}-heading`}
			bind:open={detailsOpen}
		>
			<Collapsible.Trigger class="substep-accordion-summary w-full items-center hover:no-underline">
				<div class="substep-summary-main">
					<span class="pill pill-panel">{substep.substepId}</span>
					<div class="substep-title">
						<span class="substep-title-heading" id={`substep-${domId}-heading`}>
							{substep.title}
						</span>
					</div>
				</div>
				<span class="status" aria-label={`Status: ${statusLabel}`}>{statusLabel}</span>
			</Collapsible.Trigger>
			<Collapsible.Content>
				<div class="substep-details">
					{#if substep.blockedReason || submissions.length > 0}
						<div class="substep-org-body">
							{#if substep.blockedReason}
								<p
									class="muted u-m-0 u-pre-line"
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
						<p class="muted">No data form configured for this substep</p>
					{/if}
				</div>
			</Collapsible.Content>
		</Collapsible.Root>
	{/if}
</li>
