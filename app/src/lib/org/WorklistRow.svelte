<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue, HTMLAnchorAttributes } from 'svelte/elements';

	import * as m from '#lib/paraglide/messages.js';

	import DraftBozzaMark from './DraftBozzaMark.svelte';

	type Props = {
		href?: string | null;
		ready?: boolean;
		/** Client overlay: dirty Draft beside ready status (not a Worklist TaskStatus). */
		showDraft?: boolean;
		/** Authored Job CSS color for the card's left border. */
		accentColor?: string;
		children: Snippet;
		class?: ClassValue;
	} & Omit<HTMLAnchorAttributes, 'href' | 'children' | 'class'>;

	let {
		href = null,
		ready = false,
		showDraft = false,
		accentColor,
		children,
		class: className,
		...rest
	}: Props = $props();

	const clickable = $derived(href != null && href !== '');
	const accent = $derived(accentColor?.trim() || '');
	const showMiddleChrome = $derived(showDraft || ready);
</script>

{#snippet body()}
	<div class="flex items-center gap-3">
		<div class="min-w-0 flex-1">
			{@render children()}
		</div>
		{#if showMiddleChrome}
			<div class="flex shrink-0 items-center gap-1.5">
				{#if showDraft}
					<DraftBozzaMark />
				{/if}
				{#if ready}
					<span
						class="size-2 shrink-0 rounded-full bg-emerald-500"
						role="img"
						aria-label={m.available()}
						data-testid="available-ready-dot"
					></span>
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

{#if clickable}
	<a
		href={href!}
		class={[
			'block rounded-lg border-y border-r border-zinc-200 bg-white px-4 py-3 hover:bg-zinc-50',
			accent ? 'border-l-2' : 'border-l border-zinc-200',
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
			'rounded-lg border-y border-r border-zinc-200 px-4 py-3',
			accent ? 'border-l-2' : 'border-l border-zinc-200',
			className
		]}
		style:border-left-color={accent || undefined}
		data-job-accent={accent || undefined}
		{...rest}
	>
		{@render body()}
	</div>
{/if}
