<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		/** Stable id for aria / heading target. */
		titleId?: string;
		title: string;
		/** Leading id chip (step number, task id). */
		lead?: string;
		/** Trailing muted meta (count) or status chip. */
		trail?: Snippet;
	};

	let { titleId, title, lead, trail }: Props = $props();
</script>

<span class="timeline-header-line">
	{#if lead}
		<span class="pill pill-panel timeline-header-lead">{lead}</span>
	{/if}
	<span class="timeline-header-title" id={titleId}>{title}</span>
	{#if trail}
		<span class="timeline-header-trail">
			{@render trail()}
		</span>
	{/if}
</span>
