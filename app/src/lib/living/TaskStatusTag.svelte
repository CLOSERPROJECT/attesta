<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import type { TaskStatus } from './types.js';
	import { taskStatusLabel } from './worklist-overview.js';

	type Props = {
		status: TaskStatus;
		class?: ClassValue;
	};

	let { status, class: className }: Props = $props();

	/** Experiment step-chrome `.status` tones (locked/done gray, available primary, Per te active). */
	const tone = $derived(
		status === 'actionable'
			? 'border-[#5385d1] text-[#5385d1] bg-[color-mix(in_srgb,#5385d1_5%,transparent)]'
			: status === 'available'
				? 'border-primary text-primary bg-[color-mix(in_srgb,var(--primary)_5%,transparent)]'
				: 'border-[#b3b3b3] text-[#b3b3b3] bg-[color-mix(in_srgb,#b3b3b3_5%,transparent)]'
	);
</script>

<span
	class={[
		'inline-flex min-w-[15ch] shrink-0 items-center justify-center rounded border px-2 py-0.5 text-center text-xs font-normal tracking-[0.04em]',
		tone,
		className
	]}
	data-status={status}
>
	{taskStatusLabel(status)}
</span>
