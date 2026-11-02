<script lang="ts">
	import type { CommitOutcome } from '../living/commit-port.js';
	import TaskForm from '../living/TaskForm.svelte';
	import type { TaskFormBundle } from '../living/form/task-form-bundle.js';
	import { living } from '../living/living-app.svelte.js';
	import { TaskFormError } from '../living/task-form-port.js';
	import {
		resolvePostCommitNavigation,
		resolveTaskDeepLink,
		resolveTaskExit,
		resolveUnmountableTaskTarget
	} from '../living/worklist-overview.js';
	import { route } from './index.js';
	import { navigateLivingTarget } from './navigate-target.js';

	const jobId = $derived(String(route.params.jobId ?? ''));
	const taskId = $derived(String(route.params.taskId ?? ''));
	const routeInstance = $derived(String(route.params.jobInstanceId ?? ''));
	const provisional = $derived(routeInstance === '' || routeInstance === 'new');
	const jobInstanceId = $derived(provisional ? '' : routeInstance);

	let bundle = $state.raw<TaskFormBundle | null>(null);
	let loading = $state(false);
	let error = $state.raw<string | null>(null);
	let loadGen = 0;
	let redirected = $state(false);

	function goExit() {
		navigateLivingTarget(resolveTaskExit(jobId, jobInstanceId, provisional));
	}

	function onCommitted(outcome: CommitOutcome) {
		navigateLivingTarget(
			resolvePostCommitNavigation(
				outcome.next,
				jobId,
				outcome.jobInstanceId,
				outcome.nextTaskFormBundle
			)
		);
	}

	$effect(() => {
		const jid = jobId;
		const tid = taskId;
		const isProv = provisional;
		const instanceId = jobInstanceId;
		void living.result;
		void living.loading;

		const gen = ++loadGen;
		redirected = false;

		if (!jid || !tid) {
			bundle = null;
			error = 'Missing Task address';
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
			isProv,
			living.result.worklist
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

		void living
			.fetchTaskForm({
				jobId: jid,
				taskId: tid,
				jobInstanceId: isProv ? 'new' : instanceId
			})
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
							isProv,
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
	<header class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<p class="font-mono text-xs text-zinc-500">
				{jobId} / {provisional ? 'new' : jobInstanceId} / {taskId}
			</p>
			{#if bundle}
				<h1 class="text-xl font-semibold text-zinc-900">{bundle.title}</h1>
			{:else}
				<h1 class="text-xl font-semibold text-zinc-900">Task</h1>
			{/if}
		</div>
		<button
			type="button"
			class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-700 hover:bg-zinc-50"
			data-testid="task-back-home"
			onclick={goExit}
		>
			{provisional ? 'Home' : 'Overview'}
		</button>
	</header>

	{#if living.commitError}
		<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
			Commit denied: {living.commitError}
		</p>
	{/if}

	{#if redirected}
		<p class="text-sm text-zinc-500" data-testid="task-redirecting">Redirecting…</p>
	{:else if loading}
		<p class="text-sm text-zinc-500" data-testid="task-load-status">Loading…</p>
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
			onCommitted={onCommitted}
		/>
		<button
			type="button"
			class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm text-zinc-700 hover:bg-zinc-50"
			data-testid="task-annulla"
			onclick={goExit}
		>
			Annulla
		</button>
	{/if}
</div>
