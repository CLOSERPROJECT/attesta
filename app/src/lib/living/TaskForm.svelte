<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { FailureValidationResult } from '@sjsf/form';
	import type { CommitOutcome } from './commit-port.js';
	import { living } from './living-app.svelte.js';
	import { taskPath } from './types.js';
	import ConfigForm from './form/ConfigForm.svelte';
	import HttpLinkStrip from './form/HttpLinkStrip.svelte';
	import type { TaskFormPlan } from './form/task-form-plan.js';
	import { emptySnapshot } from './form/domain-snapshot.js';
	import {
		catalogsFromBundleBindings,
		chromeDisplayFromBundleBindings,
		domainFromBundleBindings,
		findTaskFormBundle,
		planFromTaskFormBundle,
		type TaskFormBundle
	} from './form/task-form-bundle.js';
	import { LiveTaskForm } from './form/live-task-form.svelte.js';
	import { formatSubmitValidationError } from './form/submit-validation.js';

	type Props = {
		jobId: string;
		taskId: string;
		/** Empty string = provisional start (first Task / mint). */
		jobInstanceId: string;
		/** Mount payload from Load/Commit bundle set — only mount path. */
		bundle: TaskFormBundle;
		disabled?: boolean;
		/** Called after successful Commit for post-Commit routing. */
		onCommitted?: (outcome: CommitOutcome) => void;
	};

	let {
		jobId,
		taskId,
		jobInstanceId,
		bundle,
		disabled = false,
		onCommitted
	}: Props = $props();

	const path = $derived(taskPath(jobId, taskId));
	const title = $derived(bundle.title);
	const drafts = $derived(living.drafts);

	let submitError = $state.raw<string | null>(null);
	let live: LiveTaskForm | null = $state(null);
	let plan = $state.raw<TaskFormPlan | null>(null);
	let seededFromBundle = $state(false);

	function readDraft(): Record<string, unknown> {
		void drafts;
		return living.getDraft(path, jobInstanceId) ?? {};
	}

	function writeDraft(next: Record<string, unknown>) {
		living.setDraft(path, jobInstanceId, next);
	}

	/** Live Chrome/Derived `$domain` — bindings bag only. */
	function liveDomain(): unknown {
		return domainFromBundleBindings(bundle.bindings) ?? emptySnapshot();
	}

	function liveCatalogs(): unknown {
		return catalogsFromBundleBindings(bundle.bindings) ?? {};
	}

	function bundleInstance(): { id: string; label?: string } | undefined {
		const raw = bundle.bindings?.instance;
		if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
			return jobInstanceId ? { id: jobInstanceId } : undefined;
		}
		const rec = raw as Record<string, unknown>;
		const id = typeof rec.id === 'string' ? rec.id : jobInstanceId;
		if (!id) return undefined;
		const out: { id: string; label?: string } = { id };
		if (typeof rec.label === 'string' && rec.label.length > 0) out.label = rec.label;
		return out;
	}

	function onFormSubmitError(result: FailureValidationResult) {
		submitError = formatSubmitValidationError(result);
	}

	async function submitSealed(value: Record<string, unknown>) {
		submitError = null;
		const provisional = jobInstanceId === '';
		const outcome = await living.commit({
			taskPath: path,
			jobInstanceId: provisional ? '' : jobInstanceId,
			value,
			mint: provisional ? { kind: 'startingNew' } : { kind: 'none' }
		});
		if (outcome == null) return;
		const next = findTaskFormBundle(
			living.result?.taskFormBundles,
			jobId,
			taskId,
			outcome.jobInstanceId
		);
		if (next) {
			writeDraft({ ...(next.initialData ?? {}) });
		}
		onCommitted?.(outcome);
	}

	function disposeLive() {
		live?.dispose();
		live = null;
		plan = null;
	}

	function remountLive(nextPlan: TaskFormPlan) {
		disposeLive();
		plan = nextPlan;
		live = new LiveTaskForm(nextPlan, {
			getValue: () => readDraft(),
			setValue: (next) => writeDraft(next),
			getDomain: () => liveDomain(),
			getCatalogs: () => liveCatalogs(),
			getChromeDisplayBag: () => chromeDisplayFromBundleBindings(bundle.bindings),
			getInstance: () => bundleInstance(),
			onSubmit: (value) => {
				void submitSealed(value);
			},
			onSubmitError: onFormSubmitError
		});
	}

	function seedBundleDraft(next: TaskFormBundle) {
		if (seededFromBundle) return;
		const current = readDraft();
		if (Object.keys(current).length > 0) {
			seededFromBundle = true;
			return;
		}
		const initial = next.initialData ?? {};
		if (Object.keys(initial).length > 0) {
			writeDraft({ ...initial });
		}
		seededFromBundle = true;
	}

	$effect(() => {
		void drafts;
		void jobInstanceId;
		void bundle;

		seedBundleDraft(bundle);
		const compiled = planFromTaskFormBundle(bundle);
		if (plan?.projectionKey !== compiled.projectionKey || live === null) {
			remountLive(compiled);
		}
	});

	onDestroy(() => disposeLive());

	const showLinks = $derived(
		living.lastHttpLinks.length > 0 || living.lastHttpErrors.length > 0
	);
</script>

<section
	class="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-3"
	data-testid={`task-${taskId}`}
	data-task-path={path}
	data-instance-id={jobInstanceId || 'provisional'}
	data-from-bundle="1"
>
	<h3 class="text-sm font-semibold text-zinc-800">{title}</h3>

	{#if live?.form}
		{#key live.plan.projectionKey}
			<fieldset disabled={disabled || living.committing} class="space-y-4">
				<ConfigForm form={live.form} class="space-y-4" />
			</fieldset>
		{/key}
	{:else}
		<p class="text-sm text-zinc-500" data-testid="live-form-missing">Modulo non disponibile</p>
	{/if}

	{#if submitError}
		<p class="text-sm text-red-700" data-testid="task-submit-error" role="alert">
			{submitError}
		</p>
	{/if}

	{#if showLinks}
		<HttpLinkStrip items={living.lastHttpLinks} errors={living.lastHttpErrors} />
	{/if}
</section>
