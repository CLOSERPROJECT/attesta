<script lang="ts">
	import { living } from './living-app.svelte.js';
	import { lineLoadLeftoverIds } from './leftovers.js';
	import { canStartJob } from './starts-when.js';
	import {
		jobInstancesFor,
		taskPath,
		type Domain,
		type JobConfig
	} from './types.js';
	import { isTaskUnlocked, orderedTaskIds, taskSubmitted } from './unlock.js';

	type Props = {
		jobId: string;
		job: JobConfig;
		domain: Domain;
	};

	let { jobId, job, domain }: Props = $props();

	const taskIds = $derived(orderedTaskIds(job.tasks));
	const instances = $derived(jobInstancesFor(domain, jobId));
	const startLabel = $derived(job.startLabel ?? 'Nuovo');
	const drafts = $derived(living.drafts);
	const config = $derived(living.result?.config ?? null);

	const SHIFT = 'shift';
	const LINE_LOAD = 'lineLoad';
	const CONTAINER_REG = 'containerReg';
	const LOADED_EXTRACTED = 'loadedExtracted';

	const SHIFT_OPTIONS = [
		{ value: '1st', label: '1°' },
		{ value: '2nd', label: '2°' },
		{ value: '3rd', label: '3°' },
		{ value: 'whole-day', label: 'giornata intera' }
	] as const;

	const MATERIALS = [
		{ key: 'polvereDiVetro', label: 'Polvere di vetro' },
		{ key: 'granellaDiVetro', label: 'Granella di vetro' },
		{ key: 'granellaDiVetro2', label: 'Granella di vetro 2' },
		{ key: 'polvereDiSilicio', label: 'Polvere di silicio' },
		{ key: 'plastica', label: 'Plastica' },
		{ key: 'alluminio', label: 'Alluminio' },
		{ key: 'sopravaglio', label: 'Sopravaglio' },
		{ key: 'polvereDiAspirazione', label: 'Polvere di aspirazione' }
	] as const;

	type ProgressCell = 'partial' | 'done' | null | '';
	type LineLoadDraft = { progress: Record<string, ProgressCell> };

	type MaterialRow = { weight: number | ''; dualSources: boolean };
	type ContainerRegDraft = {
		sourceLoadId: string;
		secondarySourceLoadId: string;
		materials: Record<string, MaterialRow[]>;
	};

	type LoadedExtractedDraft = {
		checkedLoadedWeight: number | '';
		checkedContainerSum: number | '';
	};

	let showShift = $state(false);
	let shiftDraft = $state<string>('1st');

	const gateOpen = $derived(
		canStartJob({ job, jobId, domain, config })
	);

	const usedShifts = $derived(new Set(instances.map((i) => i.id)));
	const availableShifts = $derived(
		SHIFT_OPTIONS.filter((o) => !usedShifts.has(o.value))
	);
	const loadIds = $derived(Object.keys(domain.load ?? {}).sort());

	function openShift() {
		shiftDraft = availableShifts[0]?.value ?? '1st';
		showShift = true;
	}

	async function submitShift() {
		const id = shiftDraft;
		if (!id || usedShifts.has(id)) return;
		const ok = await living.commit({
			taskPath: taskPath(jobId, SHIFT),
			jobInstanceId: id,
			value: { shift: id }
		});
		if (ok) showShift = false;
	}

	function unlocked(taskId: string, instanceId: string): boolean {
		return isTaskUnlocked({
			orderedIds: taskIds,
			jobId,
			taskId,
			jobInstanceId: instanceId,
			submissions: domain.submissions,
			tasks: job.tasks ?? {}
		});
	}

	function done(taskId: string, instanceId: string): boolean {
		return taskSubmitted(domain, jobId, taskId, instanceId);
	}

	/** completeness some: keep form while leftovers remain even after prior Submissions. */
	function lineLoadOpen(instanceId: string): boolean {
		return unlocked(LINE_LOAD, instanceId) && lineLoadLeftoverIds(domain).length > 0;
	}

	function readLineLoad(instanceId: string): LineLoadDraft {
		void drafts;
		const leftoverIds = lineLoadLeftoverIds(domain);
		const existing = living.getDraft(taskPath(jobId, LINE_LOAD), instanceId) as
			| LineLoadDraft
			| null;
		const progress: Record<string, ProgressCell> = { ...(existing?.progress ?? {}) };
		for (const id of leftoverIds) {
			if (progress[id] !== undefined) continue;
			const cur = domain.pallet[id]?.progress;
			progress[id] = cur === 'partial' || cur === 'done' ? cur : '';
		}
		return { progress };
	}

	function setLineLoad(instanceId: string, draft: LineLoadDraft) {
		living.setDraft(
			taskPath(jobId, LINE_LOAD),
			instanceId,
			draft as unknown as Record<string, unknown>
		);
	}

	function patchProgress(instanceId: string, palletId: string, cell: ProgressCell) {
		const d = readLineLoad(instanceId);
		setLineLoad(instanceId, {
			progress: { ...d.progress, [palletId]: cell }
		});
	}

	function lineLoadBatchCount(instanceId: string): number {
		const d = readLineLoad(instanceId);
		let n = 0;
		for (const cell of Object.values(d.progress)) {
			if (cell === 'partial' || cell === 'done' || cell === null) n++;
		}
		return n;
	}

	async function submitLineLoad(instanceId: string) {
		const d = readLineLoad(instanceId);
		const leftoverIds = new Set(lineLoadLeftoverIds(domain));
		const progress: Record<string, string | null> = {};
		for (const [id, cell] of Object.entries(d.progress)) {
			if (!leftoverIds.has(id)) continue;
			if (cell === 'partial' || cell === 'done') progress[id] = cell;
			else if (cell === null) progress[id] = null;
		}
		const ok = await living.commit({
			taskPath: taskPath(jobId, LINE_LOAD),
			jobInstanceId: instanceId,
			value: { progress }
		});
		if (ok) living.clearDraft(taskPath(jobId, LINE_LOAD), instanceId);
	}

	function emptyMaterials(): Record<string, MaterialRow[]> {
		const out: Record<string, MaterialRow[]> = {};
		for (const m of MATERIALS) out[m.key] = [];
		return out;
	}

	function readContainer(instanceId: string): ContainerRegDraft {
		void drafts;
		const existing = living.getDraft(taskPath(jobId, CONTAINER_REG), instanceId) as
			| ContainerRegDraft
			| null;
		if (existing) {
			return {
				...existing,
				materials: { ...emptyMaterials(), ...existing.materials }
			};
		}
		return {
			sourceLoadId: loadIds[0] ?? '',
			secondarySourceLoadId: '',
			materials: emptyMaterials()
		};
	}

	function setContainer(instanceId: string, draft: ContainerRegDraft) {
		living.setDraft(
			taskPath(jobId, CONTAINER_REG),
			instanceId,
			draft as unknown as Record<string, unknown>
		);
	}

	function addMaterialRow(instanceId: string, key: string) {
		const d = readContainer(instanceId);
		const rows = [...(d.materials[key] ?? []), { weight: '' as number | '', dualSources: false }];
		setContainer(instanceId, {
			...d,
			materials: { ...d.materials, [key]: rows }
		});
	}

	function patchMaterialRow(
		instanceId: string,
		key: string,
		index: number,
		patch: Partial<MaterialRow>
	) {
		const d = readContainer(instanceId);
		const rows = [...(d.materials[key] ?? [])];
		rows[index] = { ...(rows[index] ?? { weight: '', dualSources: false }), ...patch };
		setContainer(instanceId, {
			...d,
			materials: { ...d.materials, [key]: rows }
		});
	}

	function anyDual(d: ContainerRegDraft): boolean {
		for (const rows of Object.values(d.materials)) {
			if (rows.some((r) => r.dualSources)) return true;
		}
		return false;
	}

	async function submitContainer(instanceId: string) {
		const d = readContainer(instanceId);
		const value: Record<string, unknown> = {
			sourceLoadId: d.sourceLoadId
		};
		if (anyDual(d) && d.secondarySourceLoadId) {
			value.secondarySourceLoadId = d.secondarySourceLoadId;
		}
		for (const m of MATERIALS) {
			const rows = (d.materials[m.key] ?? [])
				.filter((r) => r.weight !== '' && Number(r.weight) > 0)
				.map((r) => ({
					weight: Number(r.weight),
					dualSources: r.dualSources
				}));
			if (rows.length > 0) value[m.key] = rows;
		}
		await living.commit({
			taskPath: taskPath(jobId, CONTAINER_REG),
			jobInstanceId: instanceId,
			value
		});
	}

	function readLoaded(instanceId: string): LoadedExtractedDraft {
		void drafts;
		const existing = living.getDraft(taskPath(jobId, LOADED_EXTRACTED), instanceId) as
			| LoadedExtractedDraft
			| null;
		if (existing) return existing;

		let loaded: number | '' = '';
		let total = 0;
		let anyLoad = false;
		for (const facts of Object.values(domain.load)) {
			const tw = facts?.truckWeight;
			const tt = facts?.truckTare;
			if (typeof tw === 'number' && typeof tt === 'number') {
				total += tw - tt;
				anyLoad = true;
			}
		}
		if (anyLoad) loaded = total;

		let sum = 0;
		let anyC = false;
		for (const facts of Object.values(domain.container)) {
			if (typeof facts?.weight === 'number') {
				sum += facts.weight;
				anyC = true;
			}
		}
		return {
			checkedLoadedWeight: loaded,
			checkedContainerSum: anyC ? sum : ''
		};
	}

	function setLoaded(instanceId: string, draft: LoadedExtractedDraft) {
		living.setDraft(
			taskPath(jobId, LOADED_EXTRACTED),
			instanceId,
			draft as unknown as Record<string, unknown>
		);
	}

	async function submitLoaded(instanceId: string) {
		const d = readLoaded(instanceId);
		await living.commit({
			taskPath: taskPath(jobId, LOADED_EXTRACTED),
			jobInstanceId: instanceId,
			value: {
				checkedLoadedWeight: Number(d.checkedLoadedWeight),
				checkedContainerSum: Number(d.checkedContainerSum)
			}
		});
	}

	const fieldClass =
		'mt-1 w-full rounded-md border border-zinc-300 px-2 py-1.5 text-sm text-zinc-900';
	const labelClass = 'block text-xs font-medium text-zinc-600';
</script>

<section
	class="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm"
	data-testid="job-chrome"
	data-job-id={jobId}
>
	<header class="flex flex-wrap items-baseline justify-between gap-2">
		<div>
			<h2 class="text-lg font-semibold text-zinc-900">{jobId}</h2>
			{#if job.startLabel}
				<p class="mt-0.5 text-sm text-zinc-500">{job.startLabel}</p>
			{/if}
		</div>
		{#if job.repeatable}
			<span class="text-xs font-medium tracking-wide text-zinc-500 uppercase">
				repeatable
			</span>
		{/if}
	</header>

	<ul class="mt-3 flex flex-wrap gap-2" data-testid="job-task-ids">
		{#each taskIds as tid (tid)}
			<li class="rounded-md bg-zinc-100 px-2 py-0.5 font-mono text-xs text-zinc-700">
				{jobId}/{tid}
			</li>
		{/each}
	</ul>

	{#if !gateOpen}
		<p class="mt-3 text-sm text-amber-800" data-testid="line-starts-when-blocked">
			Line starts after intake palletWeight (startsWhen some).
		</p>
	{/if}

	<div class="mt-4 flex justify-end">
		<button
			type="button"
			class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
			data-testid="line-start"
			disabled={living.committing || !gateOpen || availableShifts.length === 0}
			onclick={openShift}
		>
			{startLabel}
		</button>
	</div>

	{#if showShift}
		<form
			class="mt-4 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-3"
			data-testid="task-shift"
			onsubmit={(e) => {
				e.preventDefault();
				void submitShift();
			}}
		>
			<h3 class="text-sm font-semibold text-zinc-800">shift</h3>
			<label class={labelClass}>
				Turno
				<select class={fieldClass} bind:value={shiftDraft}>
					{#each availableShifts as opt (opt.value)}
						<option value={opt.value}>{opt.label}</option>
					{/each}
				</select>
			</label>
			<div class="flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm"
					onclick={() => (showShift = false)}
				>
					Annulla
				</button>
				<button
					type="submit"
					class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
					data-testid="commit-shift"
					disabled={living.committing || !shiftDraft}
				>
					Commit
				</button>
			</div>
		</form>
	{/if}

	{#if instances.length === 0}
		<p class="mt-3 text-sm text-zinc-500" data-testid="job-instances-empty">
			No Job instances yet.
		</p>
	{:else}
		<ul class="mt-4 space-y-4" data-testid="job-instances">
			{#each instances as inst (inst.id)}
				<li
					class="rounded-md border border-zinc-200 p-3"
					data-testid="line-instance"
					data-instance-id={inst.id}
				>
					<p class="text-sm font-medium text-zinc-900">
						{inst.label || inst.id}
						<span class="font-normal text-zinc-400"> · {inst.id}</span>
					</p>

					{#if done(SHIFT, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-shift-done">
							shift · submitted
						</p>
					{/if}

					{#if lineLoadOpen(inst.id)}
						{@const ld = readLineLoad(inst.id)}
						{@const leftoverIds = lineLoadLeftoverIds(domain)}
						<form
							class="mt-3 space-y-3 border-t border-zinc-100 pt-3"
							data-testid="task-lineLoad"
							onsubmit={(e) => {
								e.preventDefault();
								void submitLineLoad(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								lineLoad · leftovers some
							</h4>
							<p class="text-xs text-zinc-500">
								Batch progress — does not unlock the next Task.
							</p>
							<div class="overflow-x-auto" data-testid="line-leftover-progress">
								<table class="min-w-full text-left text-sm">
									<thead>
										<tr class="text-xs text-zinc-500">
											<th class="py-1 pr-2">Bancale</th>
											<th class="py-1">Avanzamento</th>
										</tr>
									</thead>
									<tbody>
										{#each leftoverIds as palletId (palletId)}
											{@const cell = ld.progress[palletId] ?? ''}
											<tr data-pallet-id={palletId}>
												<td class="py-1 pr-2 font-mono text-xs">{palletId}</td>
												<td class="py-1">
													<select
														class={fieldClass}
														value={cell === null ? '__clear__' : cell}
														onchange={(e) => {
															const v = e.currentTarget.value;
															patchProgress(
																inst.id,
																palletId,
																v === ''
																	? ''
																	: v === '__clear__'
																		? null
																		: (v as 'partial' | 'done')
															);
														}}
													>
														<option value="">—</option>
														<option value="partial">partial</option>
														<option value="done">done</option>
														<option value="__clear__">clear</option>
													</select>
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
								data-testid="commit-lineLoad"
								disabled={living.committing || lineLoadBatchCount(inst.id) === 0}
							>
								Commit batch
							</button>
						</form>
					{:else if unlocked(LINE_LOAD, inst.id) && lineLoadLeftoverIds(domain).length === 0}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-lineLoad-clear">
							lineLoad · leftovers clear
						</p>
					{/if}

					{#if unlocked(CONTAINER_REG, inst.id) && !done(CONTAINER_REG, inst.id)}
						{@const cd = readContainer(inst.id)}
						<form
							class="mt-3 space-y-3 border-t border-zinc-100 pt-3"
							data-testid="task-containerReg"
							onsubmit={(e) => {
								e.preventDefault();
								void submitContainer(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								containerReg
							</h4>
							<div class="grid gap-3 sm:grid-cols-2">
								<label class={labelClass}>
									Carico sorgente
									<select
										class={fieldClass}
										value={cd.sourceLoadId}
										onchange={(e) =>
											setContainer(inst.id, {
												...cd,
												sourceLoadId: e.currentTarget.value
											})}
									>
										<option value="">—</option>
										{#each loadIds as lid (lid)}
											<option value={lid}>{lid}</option>
										{/each}
									</select>
								</label>
								{#if anyDual(cd)}
									<label class={labelClass}>
										Carico secondario
										<select
											class={fieldClass}
											value={cd.secondarySourceLoadId}
											onchange={(e) =>
												setContainer(inst.id, {
													...cd,
													secondarySourceLoadId: e.currentTarget.value
												})}
										>
											<option value="">—</option>
											{#each loadIds as lid (lid)}
												{#if lid !== cd.sourceLoadId}
													<option value={lid}>{lid}</option>
												{/if}
											{/each}
										</select>
									</label>
								{/if}
							</div>

							{#each MATERIALS as mat (mat.key)}
								{@const rows = cd.materials[mat.key] ?? []}
								<div class="space-y-2" data-material={mat.key}>
									<div class="flex items-center justify-between gap-2">
										<p class="text-xs font-medium text-zinc-700">{mat.label}</p>
										<button
											type="button"
											class="text-xs text-zinc-600 underline"
											onclick={() => addMaterialRow(inst.id, mat.key)}
										>
											+ riga
										</button>
									</div>
									{#each rows as row, idx (idx)}
										<div class="grid gap-2 sm:grid-cols-3">
											<label class={labelClass}>
												Peso
												<input
													class={fieldClass}
													type="number"
													min="0"
													step="any"
													value={row.weight}
													oninput={(e) =>
														patchMaterialRow(inst.id, mat.key, idx, {
															weight:
																e.currentTarget.value === ''
																	? ''
																	: Number(e.currentTarget.value)
														})}
												/>
											</label>
											<label class={`${labelClass} flex items-end gap-2 pb-2`}>
												<input
													type="checkbox"
													checked={row.dualSources}
													onchange={(e) =>
														patchMaterialRow(inst.id, mat.key, idx, {
															dualSources: e.currentTarget.checked
														})}
												/>
												<span>Fonti doppie</span>
											</label>
										</div>
									{/each}
								</div>
							{/each}

							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
								data-testid="commit-containerReg"
								disabled={living.committing || !cd.sourceLoadId}
							>
								Commit
							</button>
						</form>
					{:else if done(CONTAINER_REG, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-containerReg-done">
							containerReg · submitted
						</p>
					{/if}

					{#if unlocked(LOADED_EXTRACTED, inst.id) && !done(LOADED_EXTRACTED, inst.id)}
						{@const xd = readLoaded(inst.id)}
						<form
							class="mt-3 space-y-2 border-t border-zinc-100 pt-3"
							data-testid="task-loadedExtracted"
							onsubmit={(e) => {
								e.preventDefault();
								void submitLoaded(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								loadedExtracted
							</h4>
							<div class="grid gap-3 sm:grid-cols-2">
								<label class={labelClass}>
									Peso caricato verificato
									<input
										class={fieldClass}
										type="number"
										min="0"
										step="any"
										value={xd.checkedLoadedWeight}
										oninput={(e) =>
											setLoaded(inst.id, {
												...xd,
												checkedLoadedWeight:
													e.currentTarget.value === ''
														? ''
														: Number(e.currentTarget.value)
											})}
									/>
								</label>
								<label class={labelClass}>
									Somma pesi contenitori verificata
									<input
										class={fieldClass}
										type="number"
										min="0"
										step="any"
										value={xd.checkedContainerSum}
										oninput={(e) =>
											setLoaded(inst.id, {
												...xd,
												checkedContainerSum:
													e.currentTarget.value === ''
														? ''
														: Number(e.currentTarget.value)
											})}
									/>
								</label>
							</div>
							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
								data-testid="commit-loadedExtracted"
								disabled={living.committing}
							>
								Commit
							</button>
						</form>
					{:else if done(LOADED_EXTRACTED, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-loadedExtracted-done">
							loadedExtracted · submitted
						</p>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
