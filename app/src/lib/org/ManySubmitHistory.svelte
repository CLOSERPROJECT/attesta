<script lang="ts">
	import * as Accordion from '#lib/shadcn/ui/accordion/index.js';

	import TaskRecapList from './TaskRecapList.svelte';
	import type { TaskRecapRequest } from './task-recap-port.js';
	import type { TaskSubmit } from './types.js';
	import {
		formatSubmissionCount,
		manySubmitHistoryVisible
	} from './worklist-overview.js';

	type Props = {
		submit?: TaskSubmit;
		submissionCount?: number;
		request: TaskRecapRequest;
		'data-testid'?: string;
	};

	let {
		submit,
		submissionCount,
		request,
		'data-testid': testId = 'many-submit-history'
	}: Props = $props();

	let open = $state(false);

	const visible = $derived(manySubmitHistoryVisible(submit, submissionCount));
	const historyValue = $derived(
		request.groupId && request.groupInstanceId
			? `many:${request.groupId}:${request.groupInstanceId}:${request.taskId}`
			: `many:${request.taskId}`
	);
	const triggerTestId = $derived(`${testId}-trigger`);

	function onValueChange(next: string[]) {
		open = next.includes(historyValue);
	}
</script>

{#if visible}
	<Accordion.Root
		type="multiple"
		value={open ? [historyValue] : []}
		{onValueChange}
		class="flex flex-col"
		data-testid={testId}
		data-task-id={request.taskId}
	>
		<Accordion.Item
			value={historyValue}
			class="overflow-hidden rounded border border-border bg-card"
		>
			<Accordion.Trigger
				class="w-full px-3 py-2 text-sm text-muted-foreground hover:no-underline"
				data-testid={triggerTestId}
			>
				{formatSubmissionCount(submissionCount ?? 0)}
			</Accordion.Trigger>
			<Accordion.Content class="px-3 pb-3 [&_a]:no-underline">
				<TaskRecapList active={open} {request} />
			</Accordion.Content>
		</Accordion.Item>
	</Accordion.Root>
{/if}
