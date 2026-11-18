<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';
	import { Button } from '#lib/shadcn/ui/button/index.js';

	import type { CommitOutcome } from '../org/commit-port.js';
	import TaskForm from '../org/task-form.svelte';
	import type { TaskFormBundle } from '../org/form/task-form-bundle.js';
	import { orgSession } from '../org/org-app.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { TaskFormError } from '../org/task-form-port.js';
	import {
		resolveCommitFollowUp,
		resolveContinueNavigation,
		resolveTaskDeepLink,
		resolveUnmountableTaskTarget
	} from '../org/worklist-nav.js';
	import { route } from './index.js';
	import { navigateOrgTarget } from './navigate-target.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const jobId = $derived(coords.jobId);
	const taskId = $derived(coords.taskId);
	const routeInstance = $derived(coords.jobInstanceId);
	const provisionalJob = $derived(routeInstance === '' || routeInstance === 'new');
	const jobInstanceId = $derived(provisionalJob ? '' : routeInstance);
	const routeKey = $derived(`${jobId}|${routeInstance}|${taskId}`);

	let bundle = $state.raw<TaskFormBundle | null>(null);
	let loading = $state(false);
	let error = $state.raw<string | null>(null);
	let loadGen = 0;
	let redirected = $state(false);
	/** Held while stay so Continue can apply server continueNavigation. */
	let stayedOutcome = $state.raw<CommitOutcome | null>(null);
	/** Route key that owns the active stay; null during Commit→onCommitted race. */
	let stayRouteKey = $state.raw<string | null>(null);

	function endLinksStay() {
		orgSession.clearLinksStay();
		stayedOutcome = null;
		stayRouteKey = null;
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
			return;
		}
		// Drop Commit links before navigate so the next TaskForm does not inherit them.
		orgSession.clearLinksStay();
		stayedOutcome = null;
		stayRouteKey = null;
		navigateOrgTarget(followUp);
	}

	function onContinue() {
		const outcome = stayedOutcome;
		if (!outcome) return;
		endLinksStay();
		navigateOrgTarget(
			resolveContinueNavigation(
				outcome.continueNavigation,
				jobId,
				outcome.jobInstanceId,
				outcome.nextTaskFormBundle
			)
		);
	}

	$effect(() => {
		const jid = jobId;
		const tid = taskId;
		const isProvJob = provisionalJob;
		const instanceId = jobInstanceId;
		const key = routeKey;
		void orgSession.result;
		void orgSession.loading;
		void orgSession.linksStayActive;

		if (orgSession.linksStayActive) {
			if (stayRouteKey === null || stayRouteKey === key) {
				redirected = false;
				loading = false;
				error = null;
				return;
			}
			orgSession.clearLinksStay();
		}

		const gen = ++loadGen;
		redirected = false;
		stayedOutcome = null;
		stayRouteKey = null;

		if (!jid || !tid) {
			bundle = null;
			error = m.missing_task_address();
			loading = false;
			return;
		}

		if (!orgSession.result) {
			bundle = null;
			error = null;
			loading = true;
			return;
		}

		const gate = resolveTaskDeepLink(
			jid,
			instanceId,
			tid,
			isProvJob,
			orgSession.result.worklist
		);
		if (gate.kind !== 'task') {
			redirected = true;
			loading = false;
			bundle = null;
			error = null;
			navigateOrgTarget(gate);
			return;
		}

		loading = true;
		error = null;
		bundle = null;

		const fetchReq = {
			jobId: jid,
			taskId: tid,
			jobInstanceId: isProvJob ? 'new' : instanceId
		};

		void orgSession
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
					navigateOrgTarget(
						resolveUnmountableTaskTarget(
							jid,
							instanceId,
							isProvJob,
							orgSession.result?.worklist
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
	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
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
			{bundle}
			disabled={Boolean(stayedOutcome)}
			onCommitted={onCommitted}
		/>

		{#if stayedOutcome}
			<div class="space-y-3" data-testid="task-links-stay">
				<Button
					type="button"
					variant="outline"
					size="sm"
					class="bg-[var(--input)]"
					data-testid="task-links-continue"
					disabled={orgSession.committing}
					onclick={onContinue}
				>
					{m.continue()}
				</Button>
			</div>
		{/if}
	{/if}
</div>
