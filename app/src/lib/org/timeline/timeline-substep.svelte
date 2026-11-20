<script lang="ts">
	import { getBoardEmbed } from '../board/board-context.js';
	import { boardFormCache } from '../board/board-form-cache.svelte.js';
	import EmbeddedTaskPanel from '../board/embedded-task-panel.svelte';
	import { parseTaskHref } from '../board/board-url.js';
	import { taskAddressKey } from '../task-address.js';
	import TimelineChevron from './timeline-chevron.svelte';
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
		// Deep-link focus onto locked/done cards (substepId ≈ jobId.taskId).
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

	// Open when board focus lands here; do not reopen on blur (user may have collapsed).
	$effect(() => {
		if (focused) embedOpen = true;
	});

	function toggleEmbed() {
		if (!address) return;
		embedOpen = !embedOpen;
		board.setFocus(address);
	}
</script>

<li
	class={['substep', statusClass].filter(Boolean).join(' ')}
	data-org-actionable={substep.orgActionable ? 'true' : undefined}
	data-board-focused={focused ? 'true' : undefined}
>
	{#if embed && address}
		<div
			class="substep-accordion js-process-substep-panel substep-embed-host"
			data-substep-id={substep.substepId}
			data-testid="actionable-embed"
			aria-labelledby={`substep-${domId}-heading`}
			open={embedOpen ? true : undefined}
		>
			<button
				type="button"
				class="substep-accordion-summary substep-card-link w-full border-0 bg-transparent text-left"
				aria-expanded={embedOpen}
				aria-controls={`substep-${domId}-embed`}
				onclick={toggleEmbed}
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
				<span class="substep-accordion-chevron" aria-hidden="true">
					<TimelineChevron />
				</span>
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
					bind:open={embedOpen}
					onFocus={() => board.setFocus(address)}
					onCommitted={(outcome) => board.onCommitted(address, outcome)}
					onContinue={() => board.onContinue(address)}
				/>
			</div>
		</div>
	{:else}
		<details
			class="substep-accordion js-process-substep-panel"
			data-substep-id={substep.substepId}
			data-testid={focused ? 'focused-substep' : undefined}
			aria-labelledby={`substep-${domId}-heading`}
			open={focused || undefined}
		>
			<summary class="substep-accordion-summary">
				<div class="substep-summary-main">
					<span class="pill pill-panel">{substep.substepId}</span>
					<div class="substep-title">
						<span class="substep-title-heading" id={`substep-${domId}-heading`}>
							{substep.title}
						</span>
					</div>
				</div>
				<span class="status" aria-label={`Status: ${statusLabel}`}>{statusLabel}</span>
				<span class="substep-accordion-chevron" aria-hidden="true">
					<TimelineChevron />
				</span>
			</summary>
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
		</details>
	{/if}
</li>
