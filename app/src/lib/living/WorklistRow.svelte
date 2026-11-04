<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue, HTMLAnchorAttributes } from 'svelte/elements';

	type Props = {
		href?: string | null;
		ready?: boolean;
		/** Authored Job CSS color for the card's left border. */
		accentColor?: string;
		children: Snippet;
		class?: ClassValue;
	} & Omit<HTMLAnchorAttributes, 'href' | 'children' | 'class'>;

	let {
		href = null,
		ready = false,
		accentColor,
		children,
		class: className,
		...rest
	}: Props = $props();

	const clickable = $derived(href != null && href !== '');
	const accent = $derived(accentColor?.trim() || '');
</script>

{#snippet body()}
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0 flex-1">
			{@render children()}
		</div>
		{#if ready}
			<span
				class="mt-1.5 size-2 shrink-0 rounded-full bg-emerald-500"
				role="img"
				aria-label="Available"
				data-testid="available-ready-dot"
			></span>
		{/if}
	</div>
{/snippet}

{#if clickable}
	<a
		href={href!}
		class={[
			'block rounded-lg border-y border-r border-zinc-100 bg-white px-4 py-3 hover:bg-zinc-50',
			accent ? 'border-l-2' : 'border-l border-zinc-100',
			className
		]}
		style:border-left-color={accent || undefined}
		data-job-accent={accent || undefined}
		{...rest}
	>
		{@render body()}
	</a>
{:else}
	<div
		class={[
			'rounded-lg border-y border-r border-zinc-100 px-4 py-3',
			accent ? 'border-l-2' : 'border-l border-zinc-100',
			className
		]}
		style:border-left-color={accent || undefined}
		data-job-accent={accent || undefined}
		{...rest}
	>
		{@render body()}
	</div>
{/if}
