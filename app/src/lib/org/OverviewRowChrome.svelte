<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	import DraftBozzaMark from './DraftBozzaMark.svelte';
	import TaskStatusTag from './TaskStatusTag.svelte';
	import type { TaskStatus } from './types.js';

	type Props = {
		title: string;
		status: TaskStatus;
		/** 1-based Config order; omit for group occurrences and child Tasks. */
		orderNumber?: number;
		/** Client overlay: dirty Draft beside status (not a Worklist TaskStatus). */
		showDraft?: boolean;
		class?: ClassValue;
		children?: Snippet;
	};

	let {
		title,
		status,
		orderNumber,
		showDraft = false,
		class: className,
		children
	}: Props = $props();
</script>

<div class={['flex min-w-0 flex-1 items-center gap-2', className]}>
	{#if orderNumber != null}
		<span
			class="inline-flex size-5 shrink-0 items-center justify-center rounded border border-foreground bg-card font-mono text-xs leading-none text-foreground"
			data-testid="overview-order-number"
		>
			{orderNumber}
		</span>
	{/if}
	<div class="min-w-0 flex-1">
		<p class="truncate text-base font-medium leading-tight text-foreground">{title}</p>
		{@render children?.()}
	</div>
	<div class="flex shrink-0 items-center gap-1.5">
		{#if showDraft}
			<DraftBozzaMark />
		{/if}
		<TaskStatusTag {status} />
		<span
			class={[
				'inline-flex size-8 shrink-0 items-center justify-center text-muted-foreground transition-transform duration-[180ms] ease-out',
				status !== 'actionable' && 'group-aria-expanded/accordion-trigger:rotate-180',
				status === 'locked' && 'invisible'
			]}
			aria-hidden="true"
		>
			{#if status === 'actionable'}
				<ChevronRightIcon class="size-[18px]" strokeWidth={2.25} />
			{:else}
				<ChevronDownIcon class="size-[18px]" strokeWidth={2.25} />
			{/if}
		</span>
	</div>
</div>
