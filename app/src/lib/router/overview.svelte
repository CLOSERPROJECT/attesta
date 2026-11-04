<script lang="ts">
	import { onMount } from 'svelte';

	import * as m from '#lib/paraglide/messages.js';
	import * as Accordion from '#lib/shadcn/ui/accordion/index.js';
	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as Dialog from '#lib/shadcn/ui/dialog/index.js';

	import { draftAddressForTask } from '../living/drafts.js';
	import OverviewRowChrome from '../living/OverviewRowChrome.svelte';
	import TaskRecapPanel from '../living/TaskRecapPanel.svelte';
	import { living } from '../living/living-app.svelte.js';
	import type { TaskStatus } from '../living/types.js';
	import {
		formatHandoffCopy,
		jobInstanceOverview,
		provisionalSpawnDraftRowVisible,
		type OverviewTaskRow
	} from '../living/worklist-overview.js';
	import { resolvePostCommitNavigation } from '../living/worklist-nav.js';
	import { PROVISIONAL_GROUP_INSTANCE_ID } from '../living/types.js';
	import { navigate, p, route } from './index.js';
	import { navigateLivingTarget } from './navigate-target.js';

	const jobId = $derived(String(route.params.jobId ?? ''));
	const jobInstanceId = $derived(String(route.params.jobInstanceId ?? ''));

	let confirmCloseGroupId = $state.raw<string | null>(null);
	/** User-toggled accordion values; null ⇒ default (groups open, done collapsed). */
	let expandedTopOverride = $state.raw<string[] | null>(null);
	let expandedOccurrencesOverride = $state.raw<string[] | null>(null);
	let expandedChildDoneOverride = $state.raw<string[] | null>(null);
	let accordionScopeId = $state.raw('');

	onMount(() => {
		if (!living.result && !living.loading) {
			void living.hydrate();
		}
	});

	const model = $derived(
		jobInstanceOverview(living.result?.worklist, jobId, jobInstanceId)
	);
	const drafts = $derived(living.drafts);

	function leafShowDraft(taskId: string): boolean {
		void drafts;
		const addr = draftAddressForTask({ jobId, taskId, jobInstanceId });
		return living.hasDirtyDraft(addr.taskPath, addr.draftBagInstanceKey);
	}

	function childShowDraft(
		groupId: string,
		groupInstanceId: string,
		taskId: string
	): boolean {
		void drafts;
		const addr = draftAddressForTask({
			jobId,
			taskId,
			jobInstanceId,
			groupId,
			groupInstanceId
		});
		return living.hasDirtyDraft(addr.taskPath, addr.draftBagInstanceKey);
	}

	/** Dirty provisional first-child Draft while Spawn is offered — overview row, not Spawn CTA. */
	function provisionalSpawnDraftShown(row: OverviewTaskRow): boolean {
		void drafts;
		const group = row.group;
		const dirty = Boolean(
			group?.firstChildTaskId &&
				childShowDraft(row.taskId, '', group.firstChildTaskId)
		);
		return provisionalSpawnDraftRowVisible(group, dirty);
	}

	/** Provisional spawn (`new`) or any minted child occurrence has dirty Draft. */
	function groupShowDraft(row: OverviewTaskRow): boolean {
		void drafts;
		if (provisionalSpawnDraftShown(row)) return true;
		for (const gi of row.groupInstances ?? []) {
			if (gi.flatChild) {
				if (childShowDraft(row.taskId, gi.id, gi.flatChild.taskId)) return true;
				continue;
			}
			for (const child of gi.children) {
				if (childShowDraft(row.taskId, gi.id, child.taskId)) return true;
			}
		}
		return false;
	}

	const confirmCloseGroup = $derived(
		model && confirmCloseGroupId
			? (model.tasks.find((t) => t.taskId === confirmCloseGroupId)?.group ?? null)
			: null
	);

	const expandedTop = $derived.by(() => {
		if (!model) return [] as string[];
		if (accordionScopeId === model.jobInstanceId && expandedTopOverride !== null) {
			return expandedTopOverride;
		}
		return model.tasks.filter((t) => t.group).map((t) => `group:${t.taskId}`);
	});

	const expandedOccurrences = $derived.by(() => {
		if (!model) return [] as string[];
		if (accordionScopeId === model.jobInstanceId && expandedOccurrencesOverride !== null) {
			return expandedOccurrencesOverride;
		}
		return [];
	});

	const expandedChildDone = $derived.by(() => {
		if (!model) return [] as string[];
		if (accordionScopeId === model.jobInstanceId && expandedChildDoneOverride !== null) {
			return expandedChildDoneOverride;
		}
		return [];
	});

	$effect(() => {
		const id = jobInstanceId;
		if (id === PROVISIONAL_GROUP_INSTANCE_ID || id === '') {
			navigate('/');
			return;
		}
		if (living.loading || !living.result) return;
		// Finished instances stay on overview (Completed band / post-Commit deep-links).
		if (!model) {
			navigate('/');
		}
	});

	function setExpandedTop(value: string[]) {
		if (!model) return;
		accordionScopeId = model.jobInstanceId;
		expandedTopOverride = value;
	}

	function setExpandedOccurrences(value: string[]) {
		if (!model) return;
		accordionScopeId = model.jobInstanceId;
		expandedOccurrencesOverride = value;
	}

	function setExpandedChildDone(value: string[]) {
		if (!model) return;
		accordionScopeId = model.jobInstanceId;
		expandedChildDoneOverride = value;
	}

	function askClose(groupId: string) {
		confirmCloseGroupId = groupId;
	}

	function cancelClose() {
		confirmCloseGroupId = null;
	}

	async function confirmClose(groupId: string) {
		const outcome = await living.closeGroup({
			jobId,
			jobInstanceId,
			groupId
		});
		confirmCloseGroupId = null;
		if (!outcome) return;
		navigateLivingTarget(
			resolvePostCommitNavigation(
				outcome.next,
				jobId,
				outcome.jobInstanceId,
				outcome.nextTaskFormBundle
			)
		);
	}

	function onCloseDialogOpenChange(open: boolean) {
		if (!open) confirmCloseGroupId = null;
	}

	function leafTaskHref(taskId: string) {
		return p('/:jobId/:jobInstanceId/:taskId', {
			params: { jobId, jobInstanceId, taskId }
		});
	}

	function childTaskHref(groupId: string, groupInstanceId: string, taskId: string) {
		return p('/:jobId/:jobInstanceId/:groupId/:groupInstanceId/:taskId', {
			params: { jobId, jobInstanceId, groupId, groupInstanceId, taskId }
		});
	}

	function doneLeafValue(taskId: string) {
		return `done:${taskId}`;
	}

	function groupValue(groupId: string) {
		return `group:${groupId}`;
	}

	function occurrenceValue(groupId: string, instanceId: string) {
		return `occ:${groupId}:${instanceId}`;
	}

	function childDoneValue(groupId: string, instanceId: string, taskId: string) {
		return `child-done:${groupId}:${instanceId}:${taskId}`;
	}

	const rowShellBase =
		'rounded border border-border bg-card px-3 py-3 transition-colors';

	function rowHoverBorder(status: TaskStatus) {
		return status === 'locked' ? '' : 'hover:border-primary';
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
			status !== 'locked' && 'cursor-pointer'
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
		groupId?: string;
		groupInstanceId?: string;
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
	{:else if opts.linkable}
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
{/snippet}

<div class="mx-auto max-w-3xl space-y-4" data-testid="overview-route">
	{#if living.loading && !living.result}
		<p class="text-sm text-zinc-500" data-testid="overview-load-status">{m.loading()}</p>
	{:else if living.error && !living.result}
		<p class="text-sm text-red-700" data-testid="overview-load-error" role="alert">
			{living.error}
		</p>
	{:else if model}
		{#if living.commitError}
			<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
				{m.close_denied()}: {living.commitError}
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
				<div data-testid="overview-task-row" data-task-id={row.taskId}>
					{#if row.group}
						<Accordion.Item
							value={groupValue(row.taskId)}
							class={accordionItemClass(row.status)}
						>
							<Accordion.Trigger
								class={accordionTriggerClass('px-3 py-3', row.status)}
								data-testid="overview-group"
							>
								<OverviewRowChrome
									orderNumber={index + 1}
									title={row.title}
									status={row.status}
									showDraft={groupShowDraft(row)}
								/>
							</Accordion.Trigger>
							<Accordion.Content class={accordionContentClass()}>
								{@const showProvDraft = provisionalSpawnDraftShown(row)}
								{#if (row.groupInstances ?? []).length > 0}
									<Accordion.Root
										type="multiple"
										value={expandedOccurrences}
										onValueChange={setExpandedOccurrences}
										class="mb-3 flex flex-col gap-2 pt-3"
									>
										{#each row.groupInstances ?? [] as gi (gi.id)}
											{#if gi.flatChild}
												{@const flatChild = gi.flatChild}
												{@const occVal = occurrenceValue(row.taskId, gi.id)}
												<div
													data-testid="overview-group-instance"
													data-group-instance-id={gi.id}
												>
													{@render taskStatusRow({
														taskId: flatChild.taskId,
														title: gi.displayId,
														status: flatChild.status,
														linkable: flatChild.linkable,
														href: childTaskHref(
															row.taskId,
															gi.id,
															flatChild.taskId
														),
														doneValue: occVal,
														doneExpanded: expandedOccurrences.includes(occVal),
														recap: {
															jobId,
															jobInstanceId,
															taskId: flatChild.taskId,
															groupId: row.taskId,
															groupInstanceId: gi.id
														},
														showDraft: childShowDraft(
															row.taskId,
															gi.id,
															flatChild.taskId
														),
														doneItemClass: accordionItemClass(flatChild.status),
														doneTriggerClass: accordionTriggerClass(
															'px-3 py-2.5',
															flatChild.status
														),
														doneContentClass: accordionContentClass()
													})}
												</div>
											{:else}
												<Accordion.Item
													value={occurrenceValue(row.taskId, gi.id)}
													class={accordionItemClass(gi.status)}
													data-testid="overview-group-instance"
													data-group-instance-id={gi.id}
												>
													<Accordion.Trigger
														class={accordionTriggerClass('px-3 py-2.5', gi.status)}
													>
														<OverviewRowChrome title={gi.displayId} status={gi.status} />
													</Accordion.Trigger>
													<Accordion.Content class={accordionContentClass()}>
														<Accordion.Root
															type="multiple"
															value={expandedChildDone}
															onValueChange={setExpandedChildDone}
															class="flex flex-col gap-2 pt-2"
														>
															{#each gi.children as child (child.taskId)}
																{@const doneVal = childDoneValue(
																	row.taskId,
																	gi.id,
																	child.taskId
																)}
																{@render taskStatusRow({
																	taskId: child.taskId,
																	title: child.title,
																	status: child.status,
																	linkable: child.linkable,
																	href: childTaskHref(
																		row.taskId,
																		gi.id,
																		child.taskId
																	),
																	doneValue: doneVal,
																	doneExpanded:
																		expandedChildDone.includes(doneVal),
																	recap: {
																		jobId,
																		jobInstanceId,
																		taskId: child.taskId,
																		groupId: row.taskId,
																		groupInstanceId: gi.id
																	},
																	showDraft: childShowDraft(
																		row.taskId,
																		gi.id,
																		child.taskId
																	),
																	doneItemClass: accordionItemClass(child.status),
																	doneTriggerClass: accordionTriggerClass(
																		'px-3 py-2.5',
																		child.status
																	),
																	doneContentClass: accordionContentClass()
																})}
															{/each}
														</Accordion.Root>
													</Accordion.Content>
												</Accordion.Item>
											{/if}
										{/each}
									</Accordion.Root>
								{/if}

								{#if showProvDraft}
									{@const firstChildTaskId = row.group.firstChildTaskId}
									<div
										class="pt-3"
										data-testid="overview-group-instance"
										data-group-instance-id={PROVISIONAL_GROUP_INSTANCE_ID}
									>
										{@render taskStatusRow({
											taskId: firstChildTaskId,
											title: row.group.startLabel,
											status: 'actionable',
											linkable: true,
											href: childTaskHref(
												row.taskId,
												PROVISIONAL_GROUP_INSTANCE_ID,
												firstChildTaskId
											),
											doneValue: occurrenceValue(
												row.taskId,
												PROVISIONAL_GROUP_INSTANCE_ID
											),
											doneExpanded: false,
											recap: {
												jobId,
												jobInstanceId,
												taskId: firstChildTaskId,
												groupId: row.taskId,
												groupInstanceId: PROVISIONAL_GROUP_INSTANCE_ID
											},
											showDraft: true,
											doneItemClass: accordionItemClass('actionable'),
											doneTriggerClass: accordionTriggerClass(
												'px-3 py-2.5',
												'actionable'
											),
											doneContentClass: accordionContentClass()
										})}
									</div>
								{/if}

								{#if (row.group.canSpawn && !showProvDraft) || row.group.canClose}
									<div class="flex flex-wrap items-center justify-end gap-2 pt-3">
										{#if row.group.canSpawn && !showProvDraft}
											<a
												class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800"
												data-testid="overview-spawn"
												data-group-id={row.taskId}
												href={p(
													'/:jobId/:jobInstanceId/:groupId/:groupInstanceId/:taskId',
													{
														params: {
															jobId: model.jobId,
															jobInstanceId: model.jobInstanceId,
															groupId: row.taskId,
															groupInstanceId: PROVISIONAL_GROUP_INSTANCE_ID,
															taskId: row.group.firstChildTaskId
														}
													}
												)}
											>
												{row.group.startLabel}
											</a>
										{/if}
										{#if row.group.canClose}
											<button
												type="button"
												class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-800 hover:bg-zinc-50 disabled:opacity-50"
												data-testid="overview-close"
												data-group-id={row.taskId}
												disabled={living.committing}
												onclick={() => askClose(row.taskId)}
											>
												{row.group.closeLabel}
											</button>
										{/if}
									</div>
								{/if}
							</Accordion.Content>
						</Accordion.Item>
					{:else}
						{@const doneVal = doneLeafValue(row.taskId)}
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
							doneContentClass: accordionContentClass()
						})}
					{/if}
				</div>
			{/each}
		</Accordion.Root>

		<Dialog.Root
			open={confirmCloseGroupId !== null}
			onOpenChange={onCloseDialogOpenChange}
		>
			<Dialog.Content
				data-testid="overview-close-confirm"
				data-group-id={confirmCloseGroupId ?? undefined}
				showCloseButton={false}
			>
				<Dialog.Header>
					<Dialog.Title>
						{confirmCloseGroup?.closeLabel ?? m.fallback_close()}?
					</Dialog.Title>
					<Dialog.Description>
						{m.close_group_help()}
					</Dialog.Description>
				</Dialog.Header>
				<Dialog.Footer>
					<Button
						type="button"
						variant="outline"
						data-testid="overview-close-cancel"
						disabled={living.committing}
						onclick={cancelClose}
					>
						{m.cancel()}
					</Button>
					<Button
						type="button"
						data-testid="overview-close-confirm-submit"
						disabled={living.committing || !confirmCloseGroupId}
						onclick={() => {
							if (confirmCloseGroupId) void confirmClose(confirmCloseGroupId);
						}}
					>
						{confirmCloseGroup?.closeLabel ?? m.fallback_close()}
					</Button>
				</Dialog.Footer>
			</Dialog.Content>
		</Dialog.Root>
	{/if}
</div>
