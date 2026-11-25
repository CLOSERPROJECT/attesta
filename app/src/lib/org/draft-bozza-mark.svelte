<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import Pencil from '@lucide/svelte/icons/pencil';

	import * as m from '#lib/paraglide/messages.js';

	type Props = {
		class?: ClassValue;
	};

	let { class: className }: Props = $props();
</script>

<span
	class={['draft-bozza-mark', className]}
	data-testid="draft-bozza-mark"
>
	<Pencil class="draft-bozza-mark__icon" aria-hidden="true" />
	{m.draft_bozza()}
</span>

<style>
	/* Same soft-tint chip chrome as .timeline-header-trail .status; pencil replaces the status dot. */
	.draft-bozza-mark {
		display: inline-flex;
		align-items: center;
		justify-content: flex-start;
		box-sizing: border-box;
		gap: 0.35em;
		margin: 0;
		padding: 0.2rem 0.35rem;
		border: none;
		border-radius: var(--radius);
		font-family: var(--font-sans, ui-sans-serif, system-ui, sans-serif);
		font-size: 0.6875rem;
		font-weight: var(--font-weight-medium);
		letter-spacing: 0.02em;
		text-transform: none;
		line-height: 1.25;
		white-space: nowrap;
		color: var(--secondary-foreground);
		background-color: color-mix(in srgb, var(--secondary-foreground) 10%, transparent);
		flex-shrink: 0;
	}

	.draft-bozza-mark :global(.draft-bozza-mark__icon) {
		width: 0.625rem;
		height: 0.625rem;
		flex-shrink: 0;
	}
</style>
