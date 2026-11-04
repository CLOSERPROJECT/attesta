<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';

	import { livingToast } from './toast.svelte.js';
</script>

{#if livingToast.message}
	<div
		class="pointer-events-auto fixed inset-x-0 bottom-4 z-50 flex justify-center px-4"
		data-testid="living-toast"
		role="status"
	>
		<div
			class="flex max-w-md items-start gap-3 rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 shadow-md"
		>
			<p class="min-w-0 flex-1" data-testid="living-toast-message">{livingToast.message}</p>
			<button
				type="button"
				class="shrink-0 text-xs font-medium text-zinc-500 hover:text-zinc-800"
				data-testid="living-toast-dismiss"
				onclick={() => livingToast.dismiss()}
			>
				{m.toast_dismiss()}
			</button>
		</div>
	</div>
{/if}
