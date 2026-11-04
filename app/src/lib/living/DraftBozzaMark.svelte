<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import * as m from '#lib/paraglide/messages.js';

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
	{m.draft_bozza()}
</span>
