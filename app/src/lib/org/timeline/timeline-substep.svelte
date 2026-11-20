<script lang="ts">
	import TimelineChevron from './timeline-chevron.svelte';
	import TimelineSubmissions from './timeline-submissions.svelte';
	import type { TimelineSubstep } from './types.js';

	type Props = {
		substep: TimelineSubstep;
	};

	let { substep }: Props = $props();

	const domId = $derived(substep.substepId.replaceAll('.', '-'));
	const statusLabel = $derived(substep.statusLabel || substep.status);
	const cardLink = $derived(!!(substep.orgActionable && substep.href));
	const submissions = $derived(substep.submissions ?? []);
	const orgSubmissionsRow = $derived(cardLink && !!substep.submitMany && submissions.length > 0);
	const statusClass = $derived(substep.status ? `substep-${substep.status}` : '');
</script>

<li
	class={['substep', statusClass].filter(Boolean).join(' ')}
	data-org-actionable={substep.orgActionable ? 'true' : undefined}
>
	{#if cardLink}
		<a
			class="substep-card-link"
			href={substep.href}
			aria-labelledby={`substep-${domId}-heading`}
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
		</a>
		{#if orgSubmissionsRow}
			<TimelineSubmissions {submissions} />
		{/if}
	{:else}
		<details
			class="substep-accordion js-process-substep-panel"
			data-substep-id={substep.substepId}
			aria-labelledby={`substep-${domId}-heading`}
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
							<p class="muted u-m-0 u-pre-line">{substep.blockedReason}</p>
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
