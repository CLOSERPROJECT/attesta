<script lang="ts">
	import * as Collapsible from '#lib/shadcn/ui/collapsible/index.js';

	import type { TimelineSubmission } from './types.js';

	type Props = {
		submissions: TimelineSubmission[];
		/** Match /my: open nested panel when parent status is done. */
		open?: boolean;
	};

	let { submissions, open: openProp = false }: Props = $props();

	let open = $state(false);

	$effect.pre(() => {
		if (openProp) open = true;
	});
</script>

<Collapsible.Root class="substep-org-submissions" bind:open>
	<Collapsible.Trigger
		class="substep-org-submissions-summary w-full items-center hover:no-underline"
	>
		Submissions ({submissions.length})
	</Collapsible.Trigger>
	<Collapsible.Content>
		<ul class="substep-org-submissions-list">
			{#each submissions as row, i (`${row.timestamp ?? ''}-${row.userId ?? ''}-${i}`)}
				<li class="substep-org-submissions-item">
					{#if row.timestamp}
						<time datetime={row.timestamp}>{row.timestampHuman || row.timestamp}</time>
					{:else}
						<span class="muted">—</span>
					{/if}
					{#if row.userId}
						<span class="muted">· {row.userId}</span>
					{/if}
				</li>
			{/each}
		</ul>
	</Collapsible.Content>
</Collapsible.Root>
