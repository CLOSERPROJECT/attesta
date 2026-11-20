<script lang="ts">
	import type { TimelineSubmission } from './types.js';

	type Props = {
		submissions: TimelineSubmission[];
		/** Match /my: open nested details when parent status is done. */
		open?: boolean;
	};

	let { submissions, open = false }: Props = $props();
</script>

<details class="substep-org-submissions" {open}>
	<summary class="substep-org-submissions-summary">
		Submissions ({submissions.length})
	</summary>
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
</details>
