<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';
	import { Button } from '#lib/shadcn/ui/button/index.js';

	import type { CommitOutcome } from '../living/commit-port.js';
	import TaskForm from '../living/TaskForm.svelte';
	import type { TaskFormBundle } from '../living/form/task-form-bundle.js';
	import { living } from '../living/living-app.svelte.js';
	import { livingRouteCoordsFromPath } from '../living/nav-place.js';
	import { TaskFormError } from '../living/task-form-port.js';
	import { openGroupLoopChrome } from '../living/worklist-overview.js';
	import {
		resolveCommitFollowUp,
		resolveContinueNavigation,
		resolveGroupStartAnotherTarget,
		resolvePostCommitNavigation,
		resolveTaskDeepLink,
		resolveUnmountableTaskTarget
	} from '../living/worklist-nav.js';
	import { route } from './index.js';
	import { navigateLivingTarget } from './navigate-target.js';

	// Pathname segments only — sv-router leaks group* from failed 5-seg match.
	const coords = $derived(livingRouteCoordsFromPath({ pathname: route.pathname }));
	const jobId = $derived(coords.jobId);
	const taskId = $derived(coords.taskId);
	const routeInstance = $derived(coords.jobInstanceId);
	const groupId = $derived(coords.groupId);
	const groupRoute = $derived(coords.groupInstanceId);
	const provisionalJob = $derived(
		!groupId && (routeInstance === '' || routeInstance === 'new')
	);
	const provisionalSpawn = $derived(
		Boolean(groupId) && (groupRoute === '' || groupRoute === 'new')
	);
	const jobInstanceId = $derived(provisionalJob ? '' : routeInstance);
	const groupInstanceId = $derived(provisionalSpawn ? '' : groupRoute);
	const routeKey = $derived(
		`${jobId}|${routeInstance}|${groupId}|${groupRoute}|${taskId}`
	);

	let bundle = $state.raw<TaskFormBundle | null>(null);
	let loading = $state(false);
	let error = $state.raw<string | null>(null);
	let loadGen = 0;
	let redirected = $state(false);
	/** Held while stay so Continue / Close can apply server continueNavigation / navigation. */
	let stayedOutcome = $state.raw<CommitOutcome | null>(null);
	/** Route key that owns the active stay; null during Commit→onCommitted race. */
	let stayRouteKey = $state.raw<string | null>(null);
	let confirmClose = $state(false);

	const groupLoop = $derived(
		stayedOutcome && groupId
			? openGroupLoopChrome(
					living.result?.worklist,
					jobId,
					stayedOutcome.jobInstanceId || jobInstanceId,
					groupId
				)
			: null
	);

	function advanceFromClose(outcome: CommitOutcome) {
		navigateLivingTarget(
			resolvePostCommitNavigation(
				outcome.navigation === 'stay' ? 'home' : outcome.navigation,
				jobId,
				outcome.jobInstanceId,
				outcome.nextTaskFormBundle
			)
		);
	}

	function endLinksStay() {
		living.clearLinksStay();
		stayedOutcome = null;
		stayRouteKey = null;
		confirmClose = false;
	}

	function onCommitted(outcome: CommitOutcome) {
		const followUp = resolveCommitFollowUp(
			outcome.navigation,
			jobId,
			outcome.jobInstanceId,
			outcome.nextTaskFormBundle
		);
		if (followUp.kind === 'stay') {
			stayedOutcome = outcome;
			stayRouteKey = routeKey;
			confirmClose = false;
			return;
		}
		// Drop Commit links before navigate so the next TaskForm does not inherit them.
		living.clearLinksStay();
		stayedOutcome = null;
		stayRouteKey = null;
		navigateLivingTarget(followUp);
	}

	function onContinue() {
		const outcome = stayedOutcome;
		if (!outcome) return;
		endLinksStay();
		navigateLivingTarget(
			resolveContinueNavigation(
				outcome.continueNavigation,
				jobId,
				outcome.jobInstanceId,
				outcome.nextTaskFormBundle
			)
		);
	}

	function onStartAnother() {
		const outcome = stayedOutcome;
		const firstChild = groupLoop?.firstChildTaskId;
		if (!outcome || !groupId || !firstChild) return;
		endLinksStay();
		navigateLivingTarget(
			resolveGroupStartAnotherTarget(
				jobId,
				outcome.jobInstanceId,
				groupId,
				firstChild
			)
		);
	}

	function askClose() {
		confirmClose = true;
	}

	function cancelClose() {
		confirmClose = false;
	}

	async function confirmCloseGroup() {
		if (!groupId || !stayedOutcome) return;
		const outcome = await living.closeGroup({
			jobId,
			jobInstanceId: stayedOutcome.jobInstanceId,
			groupId
		});
		if (!outcome) return;
		endLinksStay();
		advanceFromClose(outcome);
	}

	$effect(() => {
		const jid = jobId;
		const tid = taskId;
		const gid = groupId;
		const giRoute = groupRoute;
		const isProvJob = provisionalJob;
		const isProvSpawn = provisionalSpawn;
		const instanceId = jobInstanceId;
		const key = routeKey;
		void living.result;
		void living.loading;
		void living.linksStayActive;

		if (living.linksStayActive) {
			if (stayRouteKey === null || stayRouteKey === key) {
				redirected = false;
				loading = false;
				error = null;
				return;
			}
			living.clearLinksStay();
		}

		const gen = ++loadGen;
		redirected = false;
		stayedOutcome = null;
		stayRouteKey = null;
		confirmClose = false;

		if (!jid || !tid) {
			bundle = null;
			error = m.missing_task_address();
			loading = false;
			return;
		}

		if (!living.result) {
			bundle = null;
			error = null;
			loading = true;
			if (!living.loading) {
				void living.hydrate();
			}
			return;
		}

		const gate = resolveTaskDeepLink(
			jid,
			instanceId,
			tid,
			isProvJob,
			living.result.worklist,
			gid,
			giRoute
		);
		if (gate.kind !== 'task') {
			redirected = true;
			loading = false;
			bundle = null;
			error = null;
			navigateLivingTarget(gate);
			return;
		}

		loading = true;
		error = null;
		bundle = null;

		const fetchReq = {
			jobId: jid,
			taskId: tid,
			jobInstanceId: isProvJob ? 'new' : instanceId,
			...(gid
				? {
						groupId: gid,
						groupInstanceId: isProvSpawn ? 'new' : giRoute
					}
				: {})
		};

		void living
			.fetchTaskForm(fetchReq)
			.then((next) => {
				if (gen !== loadGen) return;
				bundle = next;
				error = null;
			})
			.catch((err: unknown) => {
				if (gen !== loadGen) return;
				bundle = null;
				if (err instanceof TaskFormError && (err.status === 403 || err.status === 404)) {
					redirected = true;
					navigateLivingTarget(
						resolveUnmountableTaskTarget(
							jid,
							instanceId,
							isProvJob,
							living.result?.worklist
						)
					);
					return;
				}
				error =
					err instanceof TaskFormError
						? err.message
						: err instanceof Error
							? err.message
							: String(err);
			})
			.finally(() => {
				if (gen === loadGen) loading = false;
			});
	});
</script>

<div class="mx-auto max-w-3xl space-y-4" data-testid="task-route">
	{#if living.commitError}
		<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
			{stayedOutcome && confirmClose ? m.close_denied() : m.commit_denied()}: {living.commitError}
		</p>
	{/if}

	{#if redirected}
		<p class="text-sm text-zinc-500" data-testid="task-redirecting">{m.redirecting()}</p>
	{:else if loading}
		<p class="text-sm text-zinc-500" data-testid="task-load-status">{m.loading()}</p>
	{:else if error}
		<p class="text-sm text-red-700" data-testid="task-load-error" role="alert">
			{error}
		</p>
	{:else if bundle}
		<TaskForm
			{jobId}
			{taskId}
			{jobInstanceId}
			groupId={groupId}
			groupInstanceId={groupInstanceId}
			{bundle}
			disabled={Boolean(stayedOutcome)}
			onCommitted={onCommitted}
		/>

		{#if stayedOutcome}
			<div class="space-y-3" data-testid="task-links-stay">
				{#if groupId && groupLoop && (groupLoop.canSpawn || groupLoop.canClose)}
					{#if confirmClose && groupLoop.canClose}
						<div
							class="rounded-lg border border-zinc-200 bg-white px-4 py-3"
							data-testid="task-close-confirm"
							role="group"
							aria-label={m.confirm_close()}
						>
							<p class="text-sm font-medium text-zinc-900">
								{groupLoop.closeLabel}?
							</p>
							<p class="mt-0.5 text-xs text-zinc-500">
								{m.close_group_help()}
							</p>
							<div class="mt-3 flex flex-wrap gap-2">
								<button
									type="button"
									class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-800 hover:bg-zinc-50 disabled:opacity-50"
									data-testid="task-close-cancel"
									disabled={living.committing}
									onclick={cancelClose}
								>
									{m.cancel()}
								</button>
								<button
									type="button"
									class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
									data-testid="task-close-confirm-submit"
									disabled={living.committing}
									onclick={() => void confirmCloseGroup()}
								>
									{groupLoop.closeLabel}
								</button>
							</div>
						</div>
					{:else}
						<div class="flex flex-wrap gap-2">
							{#if groupLoop.canSpawn}
								<button
									type="button"
									class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
									data-testid="task-start-another"
									disabled={living.committing}
									onclick={onStartAnother}
								>
									{m.start_another()}
								</button>
							{/if}
							{#if groupLoop.canClose}
								<button
									type="button"
									class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-800 hover:bg-zinc-50 disabled:opacity-50"
									data-testid="task-close"
									disabled={living.committing}
									onclick={askClose}
								>
									{groupLoop.closeLabel}
								</button>
							{/if}
						</div>
					{/if}
				{:else}
					<Button
						type="button"
						variant="outline"
						size="sm"
						class="bg-[var(--input)]"
						data-testid="task-links-continue"
						disabled={living.committing}
						onclick={onContinue}
					>
						{m.continue()}
					</Button>
				{/if}
			</div>
		{/if}
	{/if}
</div>
