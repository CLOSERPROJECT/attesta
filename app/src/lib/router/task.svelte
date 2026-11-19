<script lang="ts">
	import * as m from '#lib/paraglide/messages.js';
	import { Button } from '#lib/shadcn/ui/button/index.js';

	import type { CommitOutcome } from '../org/commit-port.js';
	import {
		applyCommitFollowThrough,
		applyContinueFollowThrough,
		decideCommitFollowThrough,
		decideContinueFollowThrough
	} from '../org/commit-follow-through.js';
	import TaskForm from '../org/task-form.svelte';
	import { orgSession } from '../org/org-app.svelte.js';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import {
		normalizeTaskAddress,
		taskAddressKey
	} from '../org/task-address.js';
	import { taskFormMount } from '../org/task-form-mount.svelte.js';
	import { route } from './index.js';
	import { navigateOrgTarget } from './navigate-target.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const addr = $derived(normalizeTaskAddress(coords));
	const jobId = $derived(addr.jobId);
	const taskId = $derived(addr.taskId);
	const jobInstanceId = $derived(addr.jobInstanceId);
	const routeKey = $derived(taskAddressKey(addr));

	const status = $derived(taskFormMount.status);
	const bundle = $derived(taskFormMount.bundle);
	const formError = $derived(taskFormMount.error);
	const mountAddressKey = $derived(taskFormMount.addressKey);

	/** Held while stay so Continue can apply server continueNavigation. */
	let stayedOutcome = $state.raw<CommitOutcome | null>(null);
	/** Route key that owns the active stay; null during Commit→onCommitted race. */
	let stayRouteKey = $state.raw<string | null>(null);

	function clearStayUi() {
		stayedOutcome = null;
		stayRouteKey = null;
	}

	function onCommitted(outcome: CommitOutcome) {
		const decision = decideCommitFollowThrough(outcome, addr);
		if (decision.kind === 'stay') {
			stayedOutcome = outcome;
			stayRouteKey = routeKey;
			return;
		}
		clearStayUi();
		applyCommitFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: navigateOrgTarget
		});
	}

	async function onContinue() {
		const outcome = stayedOutcome;
		if (!outcome) return;
		const decision = decideContinueFollowThrough(outcome, addr);
		clearStayUi();
		await applyContinueFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: navigateOrgTarget,
			reload: async () => {
				await taskFormMount.reload();
				const worklist = orgSession.result?.worklist;
				if (!worklist) return;
				const gate = taskFormMount.applyGate(worklist);
				if (gate.kind === 'denied') {
					navigateOrgTarget(gate.redirect);
				}
			}
		});
	}

	// Clear stay if the route changes while stay was still active (e.g. sidebar nav).
	$effect(() => {
		const key = routeKey;
		if (stayRouteKey !== null && stayRouteKey !== key) {
			orgSession.clearLinksStay();
			clearStayUi();
		}
	});

	// Gate when Worklist is available for the bound address — never re-fetch.
	$effect(() => {
		const worklist = orgSession.result?.worklist;
		if (!worklist) return;
		if (mountAddressKey !== routeKey) return;

		const gate = taskFormMount.applyGate(worklist);
		if (gate.kind === 'denied') {
			navigateOrgTarget(gate.redirect);
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
