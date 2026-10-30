<script lang="ts">
	import JobPanel from './JobPanel.svelte';
	import type { Domain, JobConfig } from './types.js';

	type Props = {
		jobId: string;
		job: JobConfig;
		domain: Domain;
	};

	let { jobId, job, domain }: Props = $props();
</script>

<JobPanel {jobId} {job} {domain} />
