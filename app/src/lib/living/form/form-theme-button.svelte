<script lang="ts">
	import { type ComponentProps } from '@sjsf/form';
	import Plus from '@lucide/svelte/icons/plus';
	import { on } from 'svelte/events';

	const { children, disabled, onclick, type }: ComponentProps['button'] = $props();
</script>

<!--
  Native button + svelte/events attach (stock shadcn Button via extendByRecord never
  received our override; composeProps spreads dropped usable click wiring for array Add).
  Draft sync for in-place array mutate lives in createConfigForm — not this button.
-->
<button
	type="button"
	class="border-border bg-background hover:bg-muted hover:text-foreground shadow-xs inline-flex h-9 items-center justify-center gap-1.5 rounded-md border px-2.5 text-sm font-medium disabled:pointer-events-none disabled:opacity-50"
	{disabled}
	{@attach (node) => {
		if (!onclick) return;
		return on(node, 'click', (e) => {
			(onclick as (ev: MouseEvent) => void)(e);
		});
	}}
>
	{#if type === 'array-item-add'}
		<Plus class="size-4" />
	{/if}
	{@render children()}
</button>
