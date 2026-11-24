<script lang="ts">
	import { timelineRoleBadgeLabel } from './timeline-role-badge-label.js';
	import type { TimelineRoleBadge } from './types.js';

	type Props = {
		badges: TimelineRoleBadge[];
	};

	let { badges }: Props = $props();
</script>

<span class="timeline-role-badges">
	{#each badges as badge (badge.id)}
		{@const label = timelineRoleBadgeLabel(badge)}
		<span
			class="timeline-role-pill"
			data-role-palette={badge.palette || 'fallback'}
			data-testid="timeline-role-badge"
		>
			{label}
		</span>
	{/each}
</span>
