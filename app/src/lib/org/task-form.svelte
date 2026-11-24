<script lang="ts">
	import { onDestroy, type Snippet } from 'svelte';
	import type { FailureValidationResult } from '@sjsf/form';

	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from './commit-port.js';
	import { orgSession } from './org-app.svelte.js';
	import { draftBagInstanceKey } from './drafts.js';
	import { applyUrlSeed } from './seed-query.js';
	import { needsStartingNewMint, normalizeJobInstanceId } from './task-address.js';
	import { taskPath } from './types.js';
	import ConfigForm from './form/config-form.svelte';
	import { emptySnapshot } from './form/domain-snapshot.js';
	import {
		catalogsFromBundleBindings,
		chromeDisplayFromBundleBindings,
		domainFromBundleBindings,
		planFromTaskFormBundle,
		type TaskFormBundle
	} from './form/task-form-bundle.js';
	import { LiveTaskForm } from './form/live-task-form.svelte.js';
	import { formatSubmitValidationError } from './form/submit-validation.js';

	type Props = {
		stream: string;
		/** Stream instance; omit / empty for Play. */
		instance?: string;
		jobId: string;
		taskId: string;
		/** Empty string = provisional start (first Task / mint). */
		jobInstanceId: string;
		bundle: TaskFormBundle;
		disabled?: boolean;
		/** Provisional Start (mint.startingNew); also mint when instance is blank. */
		provisionalJob?: boolean;
		/**
		 * `?seed.*` from timeline href (or deep-link). Empty/omitted → unseeded;
		 * after URL strip, session open-seed memory still applies for provisional.
		 */
		seed?: Record<string, string>;
		/** Optional chrome left of the sjsf submit button (same row). */
		submitStart?: Snippet;
		onCommitted?: (outcome: CommitOutcome) => void;
	};

	let {
		stream,
		instance = '',
		jobId,
		taskId,
		jobInstanceId,
		bundle,
		disabled = false,
		provisionalJob = false,
		seed = {},
		submitStart,
		onCommitted
	}: Props = $props();

	const path = $derived(taskPath(jobId, taskId));
	const hrefSeed = $derived.by(() => {
		const out: Record<string, string> = {};
		for (const [k, v] of Object.entries(seed)) {
			if (k) out[k] = v;
		}
		return out;
	});
	const openSeed = $derived.by(() => {
		if (Object.keys(hrefSeed).length > 0) return hrefSeed;
		if (!provisionalJob && jobInstanceId) return {};
		void orgSession.openSeeds;
		return orgSession.getOpenSeed(path) ?? {};
	});
	const draftInstanceKey = $derived(
		draftBagInstanceKey({
			jobInstanceId,
			seed: !jobInstanceId && Object.keys(openSeed).length > 0 ? openSeed : undefined
		})
	);
	const seededFields = $derived.by(() => {
		const fromOpen = Object.keys(openSeed);
		if (fromOpen.length > 0) return fromOpen;
		return [...(orgSession.getSeededFields(path, draftInstanceKey) ?? [])];
	});
	const drafts = $derived(orgSession.drafts);
	/** Reset draft seed when task address / draft bag key / open seed changes. */
	const draftSeedKey = $derived(
		`${path}|${draftInstanceKey}|${seededFields.slice().sort().join(',')}`
	);

	let submitError = $state.raw<string | null>(null);
	let live: LiveTaskForm | null = $state(null);
	let seededFromBundle = $state(false);
	let lastSeedKey = $state.raw('');
	let lastAppliedOpenSeed = $state.raw('');

	function readDraft(): Record<string, unknown> {
		void drafts;
		return orgSession.getDraft(path, draftInstanceKey) ?? {};
	}

	function writeDraft(next: Record<string, unknown>) {
		orgSession.setDraft(path, draftInstanceKey, next);
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
		if (orgSession.committing) return;
		submitError = null;
		const mint = needsStartingNewMint({ provisionalJob, instance })
			? ({ kind: 'startingNew' } as const)
			: ({ kind: 'none' } as const);
		const outcome = await orgSession.commit({
			stream,
			instance: instance || undefined,
			taskPath: path,
			jobInstanceId: normalizeJobInstanceId(jobInstanceId),
			value,
			mint,
			draftBagInstanceKey: draftInstanceKey
		});
		if (outcome == null) return;
		onCommitted?.(outcome);
	}

	function ensureLive(compiled: ReturnType<typeof planFromTaskFormBundle>) {
		if (live === null) {
			live = new LiveTaskForm(compiled, {
				getValue: () => readDraft(),
				setValue: (next) => writeDraft(next),
				getDomain: () => liveDomain(),
				getCatalogs: () => liveCatalogs(),
				getChromeDisplayBag: () => chromeDisplayFromBundleBindings(bundle.bindings),
				getInstance: () => bundleInstance(),
				getOperator: () => orgSession.operator(),
				onSubmit: (value) => {
					void submitSealed(value);
				},
				onSubmitError: onFormSubmitError
			});
			return;
		}
		live.updatePlan(compiled);
	}

	function disposeLive() {
		live?.dispose();
		live = null;
	}

	function seedBundleDraft(next: TaskFormBundle) {
		if (seededFromBundle) return;
		// Presence (not non-empty value) — dirty {} must not be overwritten by seed.
		if (orgSession.getDraft(path, draftInstanceKey) != null) {
			seededFromBundle = true;
			return;
		}
		const initial = next.initialData ?? {};
		if (Object.keys(initial).length > 0) {
			orgSession.seedDraft(path, draftInstanceKey, { ...initial });
		}
		seededFromBundle = true;
	}

	function applyOpenSeedOnce() {
		const seedKey = Object.keys(openSeed)
			.sort()
			.map((k) => `${k}=${openSeed[k]}`)
			.join('&');
		if (seedKey === lastAppliedOpenSeed) return;
		lastAppliedOpenSeed = seedKey;
		if (Object.keys(openSeed).length > 0) {
			applyUrlSeed(openSeed, { jobId, taskId, jobInstanceId }, orgSession);
			return;
		}
		if (provisionalJob) {
			orgSession.clearOpenSeed(path);
		}
	}

	$effect(() => {
		// Do not track `drafts` here — seed/write updates the bag and would re-enter
		// ensureLive/updatePlan forever (effect_update_depth_exceeded). Form value
		// still tracks drafts via LiveTaskForm ports / syncDeepLocalToDraft.
		void draftSeedKey;
		void bundle;
		void openSeed;

		if (draftSeedKey !== lastSeedKey) {
			lastSeedKey = draftSeedKey;
			seededFromBundle = false;
			submitError = null;
			lastAppliedOpenSeed = '';
		}

		applyOpenSeedOnce();
		seedBundleDraft(bundle);
		const compiled = planFromTaskFormBundle(bundle, { seededFields });
		if (live === null || live.plan.projectionKey !== compiled.projectionKey) {
			ensureLive(compiled);
		}
	});

	onDestroy(() => disposeLive());
</script>

<section
	class="space-y-3"
	data-testid={`task-${taskId}`}
	data-task-path={path}
	data-instance-id={jobInstanceId || 'provisional'}
	data-from-bundle="1"
	data-seeded-fields={seededFields.length ? seededFields.join(',') : undefined}
>
	{#if live?.form}
		<fieldset disabled={disabled || orgSession.committing} class="space-y-4">
			<div class={submitStart ? 'relative' : undefined}>
				<ConfigForm
					form={live.form}
					class={[
						'flex flex-col gap-4 [&_button[type=submit]]:mt-6',
						submitStart
							? '[&_button[type=submit]]:ms-auto'
							: '[&_button[type=submit]]:self-end'
					]}
				/>
				{#if submitStart}
					<div
						class="absolute bottom-0 left-0 flex items-center"
						data-testid="task-form-submit-start"
					>
						{@render submitStart()}
					</div>
				{/if}
			</div>
		</fieldset>
	{:else}
		<p class="text-sm text-zinc-500" data-testid="live-form-missing">{m.form_unavailable()}</p>
	{/if}

	{#if submitError}
		<p class="text-sm text-red-700" data-testid="task-submit-error" role="alert">
			{submitError}
		</p>
	{/if}
</section>
