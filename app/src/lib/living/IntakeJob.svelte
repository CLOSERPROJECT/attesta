<script lang="ts">
	import { living } from './living-app.svelte.js';
	import { computeIntakeInstanceId } from './intake-id.js';
	import {
		derivedPanelCount,
		emptyWeighRow,
		palletWeightLeftoverIds,
		type WeighRowDraft
	} from './leftovers.js';
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
	/** Subscribe to Draft bag so instance forms re-render after setDraft. */
	const drafts = $derived(living.drafts);

	const REGISTER = 'register';
	const CODICE_FIR = 'codiceFir';
	const PALLET_WEIGHT = 'palletWeight';
	const WEIGHT_CHECK = 'weightCheck';

	type RegisterDraft = {
		client: string;
		date: string;
		wasteCode: string;
		idoneitaRiutilizzo: string;
		truckWeight: number | '';
		truckTare: number | '';
		palletCount: number | '';
		notes: string;
	};

	type CodiceFirDraft = { codiceFir: string };

	type PalletWeightDraft = {
		averagePanelWeight: number | '';
		tareKind: string;
		customTareWeight: number | '';
		weights: Record<string, WeighRowDraft>;
	};

	type WeightCheckDraft = {
		checkedTruckWeight: number | '';
		checkedPalletSum: number | '';
	};

	const today = () => new Date().toISOString().slice(0, 10);

	function defaultRegister(): RegisterDraft {
		return {
			client: 'NO',
			date: today(),
			wasteCode: '191212',
			idoneitaRiutilizzo: 'NIR',
			truckWeight: '',
			truckTare: '',
			palletCount: 1,
			notes: ''
		};
	}

	let registerDraft = $state<RegisterDraft>(defaultRegister());
	let showRegister = $state(false);

	function openRegister() {
		const saved = living.getDraft(taskPath(jobId, REGISTER), '') as RegisterDraft | null;
		registerDraft = saved ? { ...defaultRegister(), ...saved } : defaultRegister();
		showRegister = true;
	}

	const previewInstanceId = $derived(
		computeIntakeInstanceId(domain, {
			client: registerDraft.client,
			date: registerDraft.date,
			idoneitaRiutilizzo: registerDraft.idoneitaRiutilizzo
		})
	);

	function persistRegisterDraft() {
		living.setDraft(
			taskPath(jobId, REGISTER),
			'',
			{ ...registerDraft } as unknown as Record<string, unknown>
		);
	}

	async function submitRegister() {
		persistRegisterDraft();
		const id = previewInstanceId;
		if (!id) return;
		const ok = await living.commit({
			taskPath: taskPath(jobId, REGISTER),
			jobInstanceId: id,
			value: {
				client: registerDraft.client,
				date: registerDraft.date,
				wasteCode: registerDraft.wasteCode,
				idoneitaRiutilizzo: registerDraft.idoneitaRiutilizzo,
				truckWeight: Number(registerDraft.truckWeight),
				truckTare: Number(registerDraft.truckTare),
				palletCount: Number(registerDraft.palletCount),
				notes: registerDraft.notes
			}
		});
		if (ok) {
			living.clearDraft(taskPath(jobId, REGISTER), '');
			showRegister = false;
			registerDraft = defaultRegister();
		}
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

	function readCodice(instanceId: string): CodiceFirDraft {
		void drafts;
		const existing = living.getDraft(taskPath(jobId, CODICE_FIR), instanceId) as
			| CodiceFirDraft
			| null;
		return existing ?? { codiceFir: '' };
	}

	function setCodice(instanceId: string, codiceFir: string) {
		living.setDraft(taskPath(jobId, CODICE_FIR), instanceId, { codiceFir });
	}

	async function submitCodice(instanceId: string) {
		const d = readCodice(instanceId);
		await living.commit({
			taskPath: taskPath(jobId, CODICE_FIR),
			jobInstanceId: instanceId,
			value: { codiceFir: d.codiceFir.trim() }
		});
	}

	function readPallet(instanceId: string): PalletWeightDraft {
		void drafts;
		const leftoverIds = palletWeightLeftoverIds(domain, instanceId);
		const existing = living.getDraft(taskPath(jobId, PALLET_WEIGHT), instanceId) as
			| PalletWeightDraft
			| null;
		const base: PalletWeightDraft = existing ?? {
			averagePanelWeight: 10,
			tareKind: 'euro',
			customTareWeight: '',
			weights: {}
		};
		const weights = { ...base.weights };
		for (const id of leftoverIds) {
			if (!weights[id]) {
				weights[id] = emptyWeighRow({ tareKind: base.tareKind, idoneita: 'NIR' });
			}
		}
		return { ...base, weights };
	}

	function setPallet(instanceId: string, draft: PalletWeightDraft) {
		living.setDraft(
			taskPath(jobId, PALLET_WEIGHT),
			instanceId,
			draft as unknown as Record<string, unknown>
		);
	}

	function patchWeighRow(
		instanceId: string,
		palletId: string,
		patch: Partial<WeighRowDraft>
	) {
		const d = readPallet(instanceId);
		const row = { ...(d.weights[palletId] ?? emptyWeighRow()), ...patch };
		if (patch.panelCount === undefined) {
			const derived = derivedPanelCount({
				weight: row.weight,
				tareKind: row.tareKind,
				customTareWeight: d.customTareWeight,
				averagePanelWeight: d.averagePanelWeight
			});
			if (derived !== '') row.panelCount = derived;
		}
		setPallet(instanceId, {
			...d,
			weights: { ...d.weights, [palletId]: row }
		});
	}

	async function submitPalletWeight(instanceId: string) {
		const d = readPallet(instanceId);
		const weights: Record<string, Record<string, unknown>> = {};
		for (const [id, row] of Object.entries(d.weights)) {
			if (row.weight === '' || row.panelCount === '') continue;
			weights[id] = {
				weight: Number(row.weight),
				tareKind: row.tareKind,
				idoneita: row.idoneita,
				panelCount: Number(row.panelCount)
			};
		}
		const value: Record<string, unknown> = {
			averagePanelWeight: Number(d.averagePanelWeight),
			tareKind: d.tareKind,
			weights
		};
		if (d.tareKind === 'custom' && d.customTareWeight !== '') {
			value.customTareWeight = Number(d.customTareWeight);
		}
		await living.commit({
			taskPath: taskPath(jobId, PALLET_WEIGHT),
			jobInstanceId: instanceId,
			value
		});
	}

	function readWeightCheck(instanceId: string): WeightCheckDraft {
		void drafts;
		const existing = living.getDraft(taskPath(jobId, WEIGHT_CHECK), instanceId) as
			| WeightCheckDraft
			| null;
		if (existing) return existing;
		const load = domain.load[instanceId] ?? {};
		const truck = typeof load.truckWeight === 'number' ? load.truckWeight : '';
		let sum: number | '' = '';
		let total = 0;
		let any = false;
		for (const facts of Object.values(domain.pallet)) {
			if (facts?.loadId !== instanceId) continue;
			if (typeof facts.weight === 'number') {
				total += facts.weight;
				any = true;
			}
		}
		if (any) sum = total;
		return { checkedTruckWeight: truck, checkedPalletSum: sum };
	}

	function setWeightCheck(instanceId: string, draft: WeightCheckDraft) {
		living.setDraft(
			taskPath(jobId, WEIGHT_CHECK),
			instanceId,
			draft as unknown as Record<string, unknown>
		);
	}

	async function submitWeightCheck(instanceId: string) {
		const d = readWeightCheck(instanceId);
		await living.commit({
			taskPath: taskPath(jobId, WEIGHT_CHECK),
			jobInstanceId: instanceId,
			value: {
				checkedTruckWeight: Number(d.checkedTruckWeight),
				checkedPalletSum: Number(d.checkedPalletSum)
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

	<div class="mt-4 flex justify-end">
		<button
			type="button"
			class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800 disabled:opacity-50"
			data-testid="intake-start"
			disabled={living.committing}
			onclick={openRegister}
		>
			{startLabel}
		</button>
	</div>

	{#if showRegister}
		<form
			class="mt-4 space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-3"
			data-testid="task-register"
			onsubmit={(e) => {
				e.preventDefault();
				void submitRegister();
			}}
		>
			<h3 class="text-sm font-semibold text-zinc-800">register</h3>
			{#if previewInstanceId}
				<p class="font-mono text-xs text-zinc-500" data-testid="register-instance-preview">
					{previewInstanceId}
				</p>
			{/if}
			<div class="grid gap-3 sm:grid-cols-2">
				<label class={labelClass}>
					Cliente
					<select
						class={fieldClass}
						bind:value={registerDraft.client}
						onchange={persistRegisterDraft}
					>
						<option value="NO">Nord</option>
						<option value="SU">Sud</option>
					</select>
				</label>
				<label class={labelClass}>
					Data
					<input
						class={fieldClass}
						type="date"
						bind:value={registerDraft.date}
						onchange={persistRegisterDraft}
					/>
				</label>
				<label class={labelClass}>
					Codice CER
					<select
						class={fieldClass}
						bind:value={registerDraft.wasteCode}
						onchange={persistRegisterDraft}
					>
						{#each ['150101', '150102', '150103', '150104', '191201', '191204', '191212'] as code (code)}
							<option value={code}>{code}</option>
						{/each}
					</select>
				</label>
				<label class={labelClass}>
					Idoneità
					<select
						class={fieldClass}
						bind:value={registerDraft.idoneitaRiutilizzo}
						onchange={persistRegisterDraft}
					>
						<option value="NIR">NIR</option>
						<option value="IR">IR</option>
					</select>
				</label>
				<label class={labelClass}>
					Peso totale camion
					<input
						class={fieldClass}
						type="number"
						min="0"
						step="any"
						bind:value={registerDraft.truckWeight}
						onchange={persistRegisterDraft}
					/>
				</label>
				<label class={labelClass}>
					Tara camion
					<input
						class={fieldClass}
						type="number"
						min="0"
						step="any"
						bind:value={registerDraft.truckTare}
						onchange={persistRegisterDraft}
					/>
				</label>
				<label class={labelClass}>
					Numero di bancali
					<input
						class={fieldClass}
						type="number"
						min="1"
						step="1"
						bind:value={registerDraft.palletCount}
						onchange={persistRegisterDraft}
					/>
				</label>
				<label class={`${labelClass} sm:col-span-2`}>
					Note
					<input
						class={fieldClass}
						type="text"
						bind:value={registerDraft.notes}
						onchange={persistRegisterDraft}
					/>
				</label>
			</div>
			<div class="flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md border border-zinc-300 px-3 py-1.5 text-sm"
					onclick={() => (showRegister = false)}
				>
					Annulla
				</button>
				<button
					type="submit"
					class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
					data-testid="commit-register"
					disabled={living.committing || !previewInstanceId}
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
					data-testid="intake-instance"
					data-instance-id={inst.id}
				>
					<p class="text-sm font-medium text-zinc-900">
						{inst.label || inst.id}
						<span class="font-normal text-zinc-400"> · {inst.id}</span>
					</p>

					{#if unlocked(CODICE_FIR, inst.id) && !done(CODICE_FIR, inst.id)}
						{@const cd = readCodice(inst.id)}
						<form
							class="mt-3 space-y-2 border-t border-zinc-100 pt-3"
							data-testid="task-codiceFir"
							onsubmit={(e) => {
								e.preventDefault();
								void submitCodice(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								codiceFir
							</h4>
							<label class={labelClass}>
								Codice FIR
								<input
									class={fieldClass}
									type="text"
									value={cd.codiceFir}
									oninput={(e) => setCodice(inst.id, e.currentTarget.value)}
								/>
							</label>
							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
								data-testid="commit-codiceFir"
								disabled={living.committing || !cd.codiceFir.trim()}
							>
								Commit
							</button>
						</form>
					{:else if done(CODICE_FIR, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-codiceFir-done">
							codiceFir · submitted
						</p>
					{/if}

					{#if unlocked(PALLET_WEIGHT, inst.id) && !done(PALLET_WEIGHT, inst.id)}
						{@const pd = readPallet(inst.id)}
						{@const leftoverIds = palletWeightLeftoverIds(domain, inst.id)}
						<form
							class="mt-3 space-y-3 border-t border-zinc-100 pt-3"
							data-testid="task-palletWeight"
							onsubmit={(e) => {
								e.preventDefault();
								void submitPalletWeight(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								palletWeight · leftovers all
							</h4>
							<div class="grid gap-3 sm:grid-cols-3">
								<label class={labelClass}>
									Peso medio pannello (kg)
									<input
										class={fieldClass}
										type="number"
										min="0"
										step="any"
										value={pd.averagePanelWeight}
										oninput={(e) => {
											setPallet(inst.id, {
												...pd,
												averagePanelWeight:
													e.currentTarget.value === ''
														? ''
														: Number(e.currentTarget.value)
											});
										}}
									/>
								</label>
								<label class={labelClass}>
									Tara (default)
									<select
										class={fieldClass}
										value={pd.tareKind}
										onchange={(e) => {
											setPallet(inst.id, {
												...pd,
												tareKind: e.currentTarget.value
											});
										}}
									>
										<option value="euro">euro (25 kg)</option>
										<option value="industrial">industriale (30 kg)</option>
										<option value="custom">personalizzata</option>
									</select>
								</label>
								{#if pd.tareKind === 'custom'}
									<label class={labelClass}>
										Tara personalizzata (kg)
										<input
											class={fieldClass}
											type="number"
											min="0"
											step="any"
											value={pd.customTareWeight}
											oninput={(e) => {
												setPallet(inst.id, {
													...pd,
													customTareWeight:
														e.currentTarget.value === ''
															? ''
															: Number(e.currentTarget.value)
												});
											}}
										/>
									</label>
								{/if}
							</div>

							{#if leftoverIds.length === 0}
								<p class="text-sm text-zinc-500">No leftover pallets.</p>
							{:else}
								<div class="overflow-x-auto" data-testid="leftover-weights-table">
									<table class="min-w-full text-left text-sm">
										<thead>
											<tr class="text-xs text-zinc-500">
												<th class="py-1 pr-2">Bancale</th>
												<th class="py-1 pr-2">Peso</th>
												<th class="py-1 pr-2">Tara</th>
												<th class="py-1 pr-2">IR / NIR</th>
												<th class="py-1">Pannelli</th>
											</tr>
										</thead>
										<tbody>
											{#each leftoverIds as palletId (palletId)}
												{@const row = pd.weights[palletId] ?? emptyWeighRow()}
												<tr data-pallet-id={palletId}>
													<td class="py-1 pr-2 font-mono text-xs">{palletId}</td>
													<td class="py-1 pr-2">
														<input
															class={fieldClass}
															type="number"
															min="0"
															step="any"
															value={row.weight}
															oninput={(e) =>
																patchWeighRow(inst.id, palletId, {
																	weight:
																		e.currentTarget.value === ''
																			? ''
																			: Number(e.currentTarget.value)
																})}
														/>
													</td>
													<td class="py-1 pr-2">
														<select
															class={fieldClass}
															value={row.tareKind}
															onchange={(e) =>
																patchWeighRow(inst.id, palletId, {
																	tareKind: e.currentTarget.value
																})}
														>
															<option value="euro">euro</option>
															<option value="industrial">industriale</option>
															<option value="custom">custom</option>
														</select>
													</td>
													<td class="py-1 pr-2">
														<select
															class={fieldClass}
															value={row.idoneita}
															onchange={(e) =>
																patchWeighRow(inst.id, palletId, {
																	idoneita: e.currentTarget.value
																})}
														>
															<option value="NIR">NIR</option>
															<option value="IR">IR</option>
														</select>
													</td>
													<td class="py-1">
														<input
															class={fieldClass}
															type="number"
															min="0"
															step="1"
															value={row.panelCount}
															oninput={(e) =>
																patchWeighRow(inst.id, palletId, {
																	panelCount:
																		e.currentTarget.value === ''
																			? ''
																			: Number(e.currentTarget.value)
																})}
														/>
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							{/if}

							<button
								type="submit"
								class="rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
								data-testid="commit-palletWeight"
								disabled={living.committing || leftoverIds.length === 0}
							>
								Commit leftovers
							</button>
						</form>
					{:else if done(PALLET_WEIGHT, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-palletWeight-done">
							palletWeight · submitted
						</p>
					{/if}

					{#if unlocked(WEIGHT_CHECK, inst.id) && !done(WEIGHT_CHECK, inst.id)}
						{@const wd = readWeightCheck(inst.id)}
						<form
							class="mt-3 space-y-2 border-t border-zinc-100 pt-3"
							data-testid="task-weightCheck"
							onsubmit={(e) => {
								e.preventDefault();
								void submitWeightCheck(inst.id);
							}}
						>
							<h4 class="text-xs font-semibold tracking-wide text-zinc-500 uppercase">
								weightCheck
							</h4>
							<div class="grid gap-3 sm:grid-cols-2">
								<label class={labelClass}>
									Peso camion verificato
									<input
										class={fieldClass}
										type="number"
										min="0"
										step="any"
										value={wd.checkedTruckWeight}
										oninput={(e) =>
											setWeightCheck(inst.id, {
												...wd,
												checkedTruckWeight:
													e.currentTarget.value === ''
														? ''
														: Number(e.currentTarget.value)
											})}
									/>
								</label>
								<label class={labelClass}>
									Somma pesi bancali verificata
									<input
										class={fieldClass}
										type="number"
										min="0"
										step="any"
										value={wd.checkedPalletSum}
										oninput={(e) =>
											setWeightCheck(inst.id, {
												...wd,
												checkedPalletSum:
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
								data-testid="commit-weightCheck"
								disabled={living.committing}
							>
								Commit
							</button>
						</form>
					{:else if done(WEIGHT_CHECK, inst.id)}
						<p class="mt-2 text-xs text-zinc-500" data-testid="task-weightCheck-done">
							weightCheck · submitted
						</p>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
