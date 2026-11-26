<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import * as m from '#lib/paraglide/messages.js';

	type Props = {
		class?: ClassValue;
	};

	let { class: className }: Props = $props();
</script>

<span
	class={['done-by-me-mark', className]}
	data-testid="done-by-me-mark"
	aria-label={m.done_by_me()}
>
	{m.done_by_me()}
</span>

<style>
	/* Soft chip in the trail; scoped so we do not touch org-timeline.css. */
	.done-by-me-mark {
		display: inline-flex;
		align-items: center;
		justify-content: flex-start;
		box-sizing: border-box;
		margin: 0;
		padding: 0.2rem 0.35rem;
		border: none;
		border-radius: var(--radius);
		font-family: var(--font-sans, ui-sans-serif, system-ui, sans-serif);
		font-size: 0.6875rem;
		font-weight: var(--font-weight-medium);
		letter-spacing: 0.02em;
		line-height: 1.25;
		white-space: nowrap;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
		flex-shrink: 0;
	}
</style>
