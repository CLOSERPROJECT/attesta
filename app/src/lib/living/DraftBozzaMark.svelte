<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	type Props = {
		class?: ClassValue;
	};

	let { class: className }: Props = $props();
</script>

<span
	class={[
		'shrink-0 text-xs font-medium tracking-[0.04em] text-muted-foreground',
		className
	]}
	data-testid="draft-bozza-mark"
>
	Bozza
</span>
