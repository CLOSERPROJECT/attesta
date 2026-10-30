<script lang="ts">
	import JobPanel from './JobPanel.svelte';
	import type { WorklistJob } from './types.js';

	type Props = {
		jobId: string;
		worklistJob: WorklistJob;
	};

	let { jobId, worklistJob }: Props = $props();
</script>

<JobPanel {jobId} {worklistJob} />
