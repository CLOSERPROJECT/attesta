<script lang="ts">
	import type { HttpLinkItem } from './commit-port.js';
	import HttpLinkStrip from './form/HttpLinkStrip.svelte';
	import { formatFactValue } from './worklist-overview.js';

	type Props = {
		facts: Record<string, unknown>;
		titles?: Record<string, string>;
		links?: readonly HttpLinkItem[];
		'data-testid'?: string;
	};

	let {
		facts,
		titles = {},
		links = [],
		'data-testid': testId = 'submission-summary'
	}: Props = $props();

	const factEntries = $derived(
		Object.entries(facts).filter(([k]) => k.length > 0)
	);
</script>

<div class="space-y-3">
	<dl class="grid gap-2 text-sm" data-testid={testId}>
		{#each factEntries as [key, value] (key)}
			<div
				class="grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] gap-2 border-b border-zinc-100 pb-2 last:border-0"
			>
				<dt class="font-medium text-zinc-600">{titles[key] ?? key}</dt>
				<dd class="text-zinc-900">{formatFactValue(value)}</dd>
			</div>
		{/each}
	</dl>
	<HttpLinkStrip items={links} />
</div>
