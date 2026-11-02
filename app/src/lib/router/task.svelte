<script lang="ts">
	import TaskForm from '../living/TaskForm.svelte';
	import type { TaskFormBundle } from '../living/form/task-form-bundle.js';
	import { living } from '../living/living-app.svelte.js';
	import { TaskFormError } from '../living/task-form-port.js';
	import { navigate, route } from './index.js';

	const jobId = $derived(String(route.params.jobId ?? ''));
	const taskId = $derived(String(route.params.taskId ?? ''));
	const routeInstance = $derived(String(route.params.jobInstanceId ?? ''));
	const provisional = $derived(routeInstance === '' || routeInstance === 'new');
	const jobInstanceId = $derived(provisional ? '' : routeInstance);

	let bundle = $state.raw<TaskFormBundle | null>(null);
	let loading = $state(false);
	let error = $state.raw<string | null>(null);
	let loadGen = 0;

	$effect(() => {
		const jid = jobId;
		const tid = taskId;
		const inst = routeInstance;
		const isProv = inst === '' || inst === 'new';
		const instanceId = isProv ? '' : inst;
		const gen = ++loadGen;

		if (!jid || !tid) {
			bundle = null;
			error = 'Missing Task address';
			loading = false;
			return;
		}

		loading = true;
		error = null;
		bundle = null;

		if (!living.result && !living.loading) {
			void living.hydrate();
		}

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

	function goHome() {
		navigate('/');
	}

	function onCommitted(_mintedId: string) {
		navigate('/');
	}
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
			onclick={goHome}
		>
			Home
		</button>
	</header>

	{#if living.commitError}
		<p class="text-sm text-red-700" data-testid="living-commit-error" role="alert">
			Commit denied: {living.commitError}
		</p>
	{/if}

	{#if loading}
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
			onclick={goHome}
		>
			Annulla
		</button>
	{/if}
</div>
