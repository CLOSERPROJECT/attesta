<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { FailureValidationResult } from '@sjsf/form';
	import { living } from './living-app.svelte.js';
	import type { Domain } from './types.js';
	import { taskPath } from './types.js';
	import ConfigForm from './form/ConfigForm.svelte';
	import HttpLinkStrip from './form/HttpLinkStrip.svelte';
	import TableFormShell from './form/TableFormShell.svelte';
	import {
		buildLiveTaskPlanContext,
		compileTaskFormPlan,
		type TaskFormPlan
	} from './form/task-form-plan.js';
	import {
		isLeftoverTableTask,
		isMapTableTask,
		needsLeftoverProjection,
		needsSchemaProjection
	} from './form/leftover-table-task.js';
	import { LiveTaskForm } from './form/live-task-form.svelte.js';
	import {
		formatProjectFormSchemaError,
		leftoversIdentity,
		projectFormSchema,
		type ProjectedForm
	} from './form/project-form-schema.js';
	import {
		domainProgressForIds,
		progressDraftFromDomain,
		seedProgressDraftFromDomain
	} from './form/pallet-progress.js';
	import { seedMapTableDraft } from './form/map-table-mount.js';
	import { primaryDynamicProperty } from './form/dynamic-properties.js';
	import { taskTitleFromConfig } from './form/task-title.js';
	import { formatSubmitValidationError } from './form/submit-validation.js';

	type Props = {
		jobId: string;
		taskId: string;
		/** Empty string = provisional start (first Task / mint). */
		jobInstanceId: string;
		authored: unknown;
		domain: Domain;
		disabled?: boolean;
		/** Called after successful Commit (e.g. close provisional). */
		onCommitted?: (jobInstanceId: string) => void;
	};

	let {
		jobId,
		taskId,
		jobInstanceId,
		authored,
		domain,
		disabled = false,
		onCommitted
	}: Props = $props();

	const path = $derived(taskPath(jobId, taskId));
	const title = $derived(taskTitleFromConfig(authored, taskId));
	const drafts = $derived(living.drafts);

	let projected = $state.raw<ProjectedForm | null>(null);
	let projectError = $state.raw<string | null>(null);
	let submitError = $state.raw<string | null>(null);
	let projecting = $state(false);
	let live: LiveTaskForm | null = $state(null);
	let plan = $state.raw<TaskFormPlan | null>(null);
	/** Bumped after successful leftover Commit so TableFormShell remounts from domain seed. */
	let shellEpoch = $state(0);

	function readDraft(): Record<string, unknown> {
		void drafts;
		return living.getDraft(path, jobInstanceId) ?? {};
	}

	function writeDraft(next: Record<string, unknown>) {
		living.setDraft(path, jobInstanceId, next);
	}

	function liveDomain(): Record<string, unknown> {
		const out: Record<string, unknown> = {
			load: domain.load,
			pallet: domain.pallet,
			container: domain.container
		};
		if (
			authored !== null &&
			typeof authored === 'object' &&
			!Array.isArray(authored) &&
			(authored as Record<string, unknown>).tareKinds !== undefined
		) {
			out.tareKinds = (authored as Record<string, unknown>).tareKinds;
		}
		return out;
	}

	function onFormSubmitError(result: FailureValidationResult) {
		submitError = formatSubmitValidationError(result);
	}

	function resyncProgressDraftFromDomain(slice: ProjectedForm, snap: Domain) {
		const primary = primaryDynamicProperty(authored);
		if (primary !== 'progress') return;
		const domainProg = domainProgressForIds(snap, slice.leftoverIds);
		const seeded = progressDraftFromDomain(domainProg, slice.leftoverIds);
		writeDraft({ ...readDraft(), [primary]: seeded });
	}

	async function submitSealed(value: Record<string, unknown>) {
		submitError = null;
		const provisional = jobInstanceId === '';
		const mintedId = await living.commit({
			taskPath: path,
			jobInstanceId: provisional ? '' : jobInstanceId,
			value,
			mint: provisional ? { kind: 'startingNew' } : { kind: 'none' }
		});
		if (mintedId == null) return;
		if (isLeftoverTableTask(authored) && projected) {
			const fresh = living.result?.domain ?? domain;
			resyncProgressDraftFromDomain(projected, fresh);
			shellEpoch += 1;
		}
		onCommitted?.(mintedId);
	}

	function disposeLive() {
		live?.dispose();
		live = null;
		plan = null;
	}

	function remountLive(nextPlan: TaskFormPlan) {
		disposeLive();
		plan = nextPlan;
		if (nextPlan.surface !== 'live') return;
		live = new LiveTaskForm(nextPlan, {
			getValue: () => readDraft(),
			setValue: (next) => writeDraft(next),
			getDomain: () => liveDomain(),
			getSnapshot: () => domain,
			getInstance: () =>
				jobInstanceId
					? { id: jobInstanceId, label: jobInstanceId }
					: undefined,
			onSubmit: (value) => {
				void submitSealed(value);
			},
			onSubmitError: onFormSubmitError
		});
	}

	function seedProjectedDraft(slice: ProjectedForm) {
		if (!needsLeftoverProjection(authored)) return;
		let current = readDraft();
		const primary = primaryDynamicProperty(authored);
		if (primary === 'progress') {
			const domainProg = domainProgressForIds(domain, slice.leftoverIds);
			const seeded = seedProgressDraftFromDomain(
				current[primary],
				domainProg,
				slice.leftoverIds
			);
			current = { ...current, [primary]: seeded };
			writeDraft(current);
		}
		// mapTable weigh rows only — leftoverTable progress must stay enum|null|omit
		if (!isMapTableTask(authored)) return;
		const mapSeeded = seedMapTableDraft(current, authored, slice.leftoverIds);
		if (mapSeeded) writeDraft(mapSeeded);
	}

	async function refreshProjection() {
		if (!needsSchemaProjection(authored)) {
			projected = null;
			projectError = null;
			return;
		}
		projecting = true;
		try {
			const result = await projectFormSchema(domain, authored, {
				instance: jobInstanceId
					? { id: jobInstanceId, label: jobInstanceId }
					: undefined
			});
			if (!result.ok) {
				projected = null;
				projectError = formatProjectFormSchemaError(result.error);
				return;
			}
			projected = result.value;
			projectError = null;
			seedProjectedDraft(result.value);
		} finally {
			projecting = false;
		}
	}

	$effect(() => {
		void domain;
		void authored;
		void jobInstanceId;
		void refreshProjection();
	});

	$effect(() => {
		void drafts;
		void projected;
		void authored;

		if (isLeftoverTableTask(authored)) {
			disposeLive();
			return;
		}

		const built = buildLiveTaskPlanContext(authored, projected);
		if (built.kind === 'skip' || built.kind === 'awaiting-projection') {
			disposeLive();
			return;
		}
		const compiled = compileTaskFormPlan(authored, built.ctx);
		if ('error' in compiled) {
			disposeLive();
			projectError = compiled.error;
			return;
		}
		if (plan?.projectionKey !== compiled.projectionKey || live === null) {
			remountLive(compiled);
		}
	});

	onDestroy(() => disposeLive());

	const leftoverIds = $derived(projected?.leftoverIds ?? []);
	const projectionKey = $derived(
		projected ? leftoversIdentity(projected.leftoverIds) : ''
	);
	const shellKey = $derived(`${projectionKey}:${shellEpoch}`);
	const showLinks = $derived(
		living.lastHttpLinks.length > 0 || living.lastHttpErrors.length > 0
	);
</script>

<section
	class="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-3"
	data-testid={`task-${taskId}`}
	data-task-path={path}
	data-instance-id={jobInstanceId || 'provisional'}
>
	<h3 class="text-sm font-semibold text-zinc-800">{title}</h3>

	{#if projectError}
		<p class="text-sm text-red-700" data-testid="task-project-error">{projectError}</p>
	{:else if projecting && needsSchemaProjection(authored)}
		<p class="text-sm text-zinc-500">Loading schema…</p>
	{:else if isLeftoverTableTask(authored)}
		{#if projected}
			{#key shellKey}
				<fieldset disabled={disabled || living.committing} class="space-y-4">
					<TableFormShell
						schema={projected.schema}
						uiSchema={projected.uiSchema}
						leftoverIds={[...projected.leftoverIds]}
						columns={projected.columns}
						valueType="progress"
						value={readDraft()}
						onValueChange={writeDraft}
						onSubmit={(value) => void submitSealed(value)}
						onSubmitError={onFormSubmitError}
						disabled={disabled || living.committing}
					/>
				</fieldset>
			{/key}
			<p class="text-xs text-zinc-500">
				Residui: {leftoverIds.length ? leftoverIds.join(', ') : 'nessuno'}
			</p>
		{/if}
	{:else if live?.form}
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
