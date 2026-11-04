<script lang="ts">
	import type { HttpLinkItem } from '../commit-port.js';

	type Props = {
		items: readonly HttpLinkItem[];
		errors?: readonly string[];
	};

	let { items, errors = [] }: Props = $props();

	function openLink(item: HttpLinkItem) {
		if (item.action === 'download') {
			const a = document.createElement('a');
			a.href = item.url;
			a.download = item.filename ?? 'download';
			a.rel = 'noopener';
			a.target = '_blank';
			a.click();
			return;
		}
		window.open(item.url, '_blank', 'noopener,noreferrer');
	}
</script>

{#if items.length > 0}
	<div class="flex flex-wrap gap-2" data-testid="task-http-links">
		{#each items as item, index (`${item.label}:${item.url}:${index}`)}
			<button
				type="button"
				class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
				data-testid="task-http-link"
				onclick={() => openLink(item)}
			>
				{item.label}
			</button>
		{/each}
	</div>
{/if}
{#if errors.length > 0}
	<ul class="space-y-1 text-sm text-amber-800" data-testid="task-http-errors">
		{#each errors as err, index (`${err}:${index}`)}
			<li>{err}</li>
		{/each}
	</ul>
{/if}
