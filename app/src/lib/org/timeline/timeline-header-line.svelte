<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		/** Stable id for aria / heading target. */
		titleId?: string;
		title: string;
		/** Hierarchical outline mark: 2, 2A, 2A.1 */
		mark: string;
		/** Trailing status chip (tasks). */
		trail?: Snippet;
	};

	let { titleId, title, mark, trail }: Props = $props();
</script>

<span class="timeline-header-line">
	<span class="timeline-header-index">{mark}</span>
	<span class="timeline-header-title" id={titleId}>{title}</span>
	{#if trail}
		<span class="timeline-header-trail">
			{@render trail()}
		</span>
	{/if}
</span>
