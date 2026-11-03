<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue, HTMLAnchorAttributes } from 'svelte/elements';

	type Props = {
		href?: string | null;
		ready?: boolean;
		children: Snippet;
		class?: ClassValue;
	} & Omit<HTMLAnchorAttributes, 'href' | 'children' | 'class'>;

	let { href = null, ready = false, children, class: className, ...rest }: Props = $props();

	const clickable = $derived(href != null && href !== '');
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
			'block rounded-lg border border-zinc-100 bg-white px-4 py-3 hover:bg-zinc-50',
			className
		]}
		{...rest}
	>
		{@render body()}
	</a>
{:else}
	<div class={['rounded-lg border border-zinc-100 px-4 py-3', className]} {...rest}>
		{@render body()}
	</div>
{/if}
