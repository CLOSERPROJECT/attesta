<script lang="ts">
	import type { ClassValue } from 'svelte/elements';

	import { progressFraction } from './worklist-home.js';

	type Props = {
		jobId: string;
		jobTitle: string;
		label: string;
		mode: 'start' | 'progress';
		size?: 'sm' | 'xs';
		done?: number;
		total?: number;
		progressTestId?: string;
		class?: ClassValue;
	};

	const RING = {
		sm: { box: 12, r: 4.5, stroke: 1.5, svgClass: 'size-3', countClass: 'text-[11px]' },
		xs: { box: 10, r: 3.75, stroke: 1.25, svgClass: 'size-2.5', countClass: 'text-[10px]' }
	} as const;

	let {
		jobId,
		jobTitle,
		label,
		mode,
		size = 'sm',
		done = 0,
		total = 0,
		progressTestId,
		class: className
	}: Props = $props();

	const textClass = $derived(size === 'xs' ? 'text-xs' : 'text-sm');
	const ring = $derived(RING[size]);
	const ringC = $derived(2 * Math.PI * ring.r);
	const ringOffset = $derived(ringC * (1 - progressFraction(done, total)));
	const center = $derived(ring.box / 2);
</script>

{#snippet jobTitleMark()}
	{#if jobTitle === jobId}
		<span class="font-mono font-semibold">{jobId}</span>
	{:else}
		<span class="font-semibold">{jobTitle}</span>
	{/if}
{/snippet}

{#if mode === 'progress'}
	<div class={['flex min-w-0 items-center gap-1.5', className]}>
		<p class={['min-w-0 truncate text-zinc-900', textClass]}>
			{@render jobTitleMark()}
			<span class="text-zinc-400">·</span>
			<span class="font-normal text-zinc-600">{label}</span>
		</p>
		<span
			class="inline-flex shrink-0 items-center gap-1"
			role="img"
			aria-label="{done} of {total} tasks done"
			data-testid={progressTestId}
			data-done={done}
			data-total={total}
		>
			<svg
				class={['-rotate-90 text-zinc-200', ring.svgClass]}
				viewBox="0 0 {ring.box} {ring.box}"
				aria-hidden="true"
			>
				<circle
					cx={center}
					cy={center}
					r={ring.r}
					fill="none"
					stroke="currentColor"
					stroke-width={ring.stroke}
				/>
				<circle
					class="text-zinc-700"
					cx={center}
					cy={center}
					r={ring.r}
					fill="none"
					stroke="currentColor"
					stroke-width={ring.stroke}
					stroke-linecap="round"
					stroke-dasharray={ringC}
					stroke-dashoffset={ringOffset}
				/>
			</svg>
			<span class={['tabular-nums leading-none text-zinc-500', ring.countClass]}>
				{done}/{total}
			</span>
		</span>
	</div>
{:else}
	<p class={['min-w-0 text-zinc-900', textClass, className]}>
		{@render jobTitleMark()}
		<span class="text-zinc-400">·</span>
		<span class="font-normal text-zinc-600">{label}</span>
	</p>
{/if}
