<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';
	import * as Accordion from '#lib/shadcn/ui/accordion/index.js';

	import { draftAddressForTask } from '../org/drafts.js';
	import ManySubmitHistory from '../org/many-submit-history.svelte';
	import OverviewRowChrome from '../org/overview-row-chrome.svelte';
	import TaskRecapPanel from '../org/task-recap-panel.svelte';
	import { orgSession } from '../org/org-app.svelte.js';
	import type { TaskStatus, TaskSubmit } from '../org/types.js';
	import { PROVISIONAL_JOB_INSTANCE_ID } from '../org/types.js';
	import {
		formatHandoffCopy,
		jobInstanceOverview
	} from '../org/worklist-overview.js';
	import { navigate, p, route } from './index.js';

	const jobId = $derived(String(route.params.jobId ?? ''));
	const jobInstanceId = $derived(String(route.params.jobInstanceId ?? ''));

	/** User-toggled accordion values; null ⇒ default (done collapsed). */
	let expandedTopOverride = $state.raw<string[] | null>(null);
	let accordionScopeId = $state.raw('');

	onMount(() => {
		if (!orgSession.result && !orgSession.loading) {
			void orgSession.hydrate();
		}
	});

	const model = $derived(
		jobInstanceOverview(orgSession.result?.worklist, jobId, jobInstanceId)
	);
	const drafts = $derived(orgSession.drafts);

	function leafShowDraft(taskId: string): boolean {
		void drafts;
		const addr = draftAddressForTask({ jobId, taskId, jobInstanceId });
		return orgSession.hasDirtyDraft(addr.taskPath, addr.draftBagInstanceKey);
	}

	const expandedTop = $derived.by(() => {
		if (!model) return [] as string[];
		if (accordionScopeId === model.jobInstanceId && expandedTopOverride !== null) {
			return expandedTopOverride;
		}
		return [];
	});

	$effect(() => {
		const id = jobInstanceId;
		if (id === PROVISIONAL_JOB_INSTANCE_ID || id === '') {
			navigate('/');
			return;
		}
		if (orgSession.loading || !orgSession.result) return;
		// Finished instances stay on overview (Archive rediscovery / deep-links).
		if (!model) {
			navigate('/');
		}
	});

	function setExpandedTop(value: string[]) {
		if (!model) return;
		accordionScopeId = model.jobInstanceId;
		expandedTopOverride = value;
	}

	function leafTaskHref(taskId: string) {
		return p('/:jobId/:jobInstanceId/:taskId', {
			params: { jobId, jobInstanceId, taskId }
		});
	}

	function doneLeafValue(taskId: string) {
		return `done:${taskId}`;
	}

	const rowShellBase =
		'rounded border border-border bg-card px-3 py-3 transition-colors';

	function rowHoverBorder(status: TaskStatus) {
		return status === 'unavailable' || status === 'done' ? '' : 'hover:border-primary';
	}

	function rowShellClass(status: TaskStatus) {
		return [rowShellBase, rowHoverBorder(status)].filter(Boolean).join(' ');
	}

	function linkShellClass(status: TaskStatus) {
		return `${rowShellClass(status)} block`;
	}

	function accordionItemClass(status: TaskStatus, extra = '') {
		return [
			'overflow-hidden rounded border border-border bg-card transition-colors',
			rowHoverBorder(status),
			extra
		]
			.filter(Boolean)
			.join(' ')
			.trim();
	}

	function accordionTriggerClass(padding = 'px-3 py-3', status?: TaskStatus) {
		return [
			'w-full',
			padding,
			'items-center hover:no-underline **:data-[slot=accordion-trigger-icon]:hidden',
			status !== 'unavailable' && status !== 'done' && 'cursor-pointer'
		]
			.filter(Boolean)
			.join(' ');
	}

	// Cancels AccordionContent's prose `[&_a]:underline` for overview chrome links/CTAs.
	function accordionContentClass(extra = 'px-3 pb-3') {
		return [extra, '[&_a]:no-underline'].filter(Boolean).join(' ');
	}

	type TaskRowRecap = {
		jobId: string;
		jobInstanceId: string;
		taskId: string;
	};

	type TaskRowOpts = {
		taskId: string;
		title: string;
		status: TaskStatus;
		linkable: boolean;
		href: string;
		doneValue: string;
		doneExpanded: boolean;
		recap: TaskRowRecap;
		orderNumber?: number;
		showDraft?: boolean;
		doneItemClass: string;
		doneTriggerClass: string;
		doneContentClass: string;
		submit?: TaskSubmit;
		submissionCount?: number;
	};
</script>

{#snippet taskStatusRow(opts: TaskRowOpts)}
	{#if opts.status === 'done'}
		<Accordion.Item value={opts.doneValue} class={opts.doneItemClass}>
			<Accordion.Trigger
				class={opts.doneTriggerClass}
				data-testid="overview-task-status"
			>
				<OverviewRowChrome
					orderNumber={opts.orderNumber}
					title={opts.title}
					status={opts.status}
					showDraft={opts.showDraft}
				/>
			</Accordion.Trigger>
			<Accordion.Content class={opts.doneContentClass}>
				<TaskRecapPanel active={opts.doneExpanded} request={opts.recap} />
			</Accordion.Content>
		</Accordion.Item>
	{:else}
		<div class="flex flex-col gap-2">
			{#if opts.linkable}
				<a
					class={linkShellClass(opts.status)}
					data-testid="overview-task-link"
					data-task-id={opts.taskId}
					href={opts.href}
				>
					<OverviewRowChrome
						orderNumber={opts.orderNumber}
						title={opts.title}
						status={opts.status}
						showDraft={opts.showDraft}
					/>
				</a>
			{:else}
				<div
					class={rowShellClass(opts.status)}
					data-testid="overview-task-status"
					data-task-id={opts.taskId}
				>
					<OverviewRowChrome
						orderNumber={opts.orderNumber}
						title={opts.title}
						status={opts.status}
						showDraft={opts.showDraft}
					/>
				</div>
			{/if}
			<ManySubmitHistory
				submit={opts.submit}
				submissionCount={opts.submissionCount}
				request={opts.recap}
				data-testid="overview-many-history"
			/>
		</div>
	{/if}
{/snippet}

<div class="mx-auto max-w-3xl space-y-4" data-testid="overview-route">
	{#if orgSession.loading && !orgSession.result}
		<p class="text-sm text-zinc-500" data-testid="overview-load-status">{m.loading()}</p>
	{:else if orgSession.error && !orgSession.result}
		<p class="text-sm text-red-700" data-testid="overview-load-error" role="alert">
			{orgSession.error}
		</p>
	{:else if model}
		{#if orgSession.commitError}
			<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
				{m.commit_denied()}: {orgSession.commitError}
			</p>
		{/if}

		{#if model.handoffRoles}
			<p
				class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900"
				data-testid="overview-handoff"
			>
				{formatHandoffCopy(model.handoffRoles)}
			</p>
		{/if}

		<Accordion.Root
			type="multiple"
			value={expandedTop}
			onValueChange={setExpandedTop}
			class="flex flex-col gap-2"
			data-testid="overview-tasks"
		>
			{#each model.tasks as row, index (row.taskId)}
				{@const doneVal = doneLeafValue(row.taskId)}
				<div data-testid="overview-task-row" data-task-id={row.taskId}>
					{@render taskStatusRow({
						taskId: row.taskId,
						title: row.title,
						status: row.status,
						linkable: row.linkable,
						href: leafTaskHref(row.taskId),
						doneValue: doneVal,
						doneExpanded: expandedTop.includes(doneVal),
						recap: {
							jobId: model.jobId,
							jobInstanceId: model.jobInstanceId,
							taskId: row.taskId
						},
						orderNumber: index + 1,
						showDraft: leafShowDraft(row.taskId),
						doneItemClass: accordionItemClass(row.status),
						doneTriggerClass: accordionTriggerClass('px-3 py-3', row.status),
						doneContentClass: accordionContentClass(),
						submit: row.submit,
						submissionCount: row.submissionCount
					})}
				</div>
			{/each}
		</Accordion.Root>
	{/if}
</div>
