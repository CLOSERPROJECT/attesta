<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';
	import { Button } from '#lib/shadcn/ui/button/index.js';

	import type { CommitOutcome } from '../org/commit-port.js';
	import TaskForm from '../org/task-form.svelte';
	import { orgSession } from '../org/org-app.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { taskFormMount } from '../org/task-form-mount.svelte.js';
	import { resolveCommitFollowUp, resolveContinueNavigation } from '../org/worklist-nav.js';
	import { route } from './index.js';
	import { navigateOrgTarget } from './navigate-target.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const jobId = $derived(coords.jobId);
	const taskId = $derived(coords.taskId);
	const routeInstance = $derived(coords.jobInstanceId);
	const provisionalJob = $derived(routeInstance === '' || routeInstance === 'new');
	const jobInstanceId = $derived(provisionalJob ? '' : routeInstance);
	const routeKey = $derived(`${jobId}|${routeInstance}|${taskId}`);

	const status = $derived(taskFormMount.status);
	const bundle = $derived(taskFormMount.bundle);
	const formError = $derived(taskFormMount.error);

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

	// Clear stay if the route changes while stay was still active (e.g. sidebar nav).
	$effect(() => {
		const key = routeKey;
		if (stayRouteKey !== null && stayRouteKey !== key) {
			endLinksStay();
		}
	});
</script>

<div class="mx-auto max-w-3xl space-y-4" data-testid="task-route">
	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
		</p>
	{/if}

	{#if status === 'denied'}
		<p class="text-sm text-zinc-500" data-testid="task-redirecting">{m.redirecting()}</p>
	{:else if status === 'loading' || status === 'idle'}
		<p class="text-sm text-zinc-500" data-testid="task-load-status">{m.loading()}</p>
	{:else if status === 'error'}
		<p class="text-sm text-red-700" data-testid="task-load-error" role="alert">
			{formError}
		</p>
	{:else if status === 'ready' && bundle}
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
