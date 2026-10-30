<script lang="ts">
	import {
		getComponent,
		getFieldErrors,
		getFormContext,
		type ComponentProps
	} from '@sjsf/form';
	import type { SchemaObjectValue } from '@sjsf/form/core';
	import { getLeftoverContext } from './leftover-context.js';
	import { isPalletProgress, type PalletProgress } from './pallet-progress.js';

	let {
		config,
		value = $bindable(),
		uiOption
	}: ComponentProps['leftoverTableField'] = $props();

	const ctx = getFormContext();
	const leftovers = getLeftoverContext();

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));

	const rows = $derived(leftovers.ids);

	// Display SoT is sjsf value (host seeds Draft from domain at open).
	// null = clear write; missing/non-progress → undefined (available).
	const displayById = $derived.by(() => {
		const out: Record<string, PalletProgress | undefined> = {};
		for (const id of rows) {
			const cell = value?.[id];
			if (cell === null) out[id] = undefined;
			else if (isPalletProgress(cell)) out[id] = cell;
			else out[id] = undefined;
		}
		return out;
	});

	function cellNumber(id: string): string {
		const current = value?.[id];
		if (typeof current === 'number' && Number.isFinite(current)) return String(current);
		if (typeof current === 'string') return current;
		return '';
	}

	function commitFieldValue(next: SchemaObjectValue): void {
		value = next;
	}

	function setNumberCell(id: string, raw: string): void {
		const next: SchemaObjectValue = { ...(value ?? {}) };
		const trimmed = raw.trim();
		if (trimmed === '') {
			delete next[id];
		} else {
			const parsed = Number(trimmed);
			next[id] = Number.isFinite(parsed) ? parsed : trimmed;
		}
		commitFieldValue(next);
	}

	function setProgressCell(id: string, nextProgress: PalletProgress | undefined): void {
		const next: SchemaObjectValue = { ...(value ?? {}) };
		// `null` = clear intent for submit (unset domain progress); omit = untouched.
		if (nextProgress === undefined) next[id] = null;
		else next[id] = nextProgress;
		commitFieldValue(next);
	}
</script>

<Template
	type="template"
	showTitle={true}
	useLabel={false}
	widgetType="textWidget"
	{uiOption}
	value={undefined}
	{config}
	{errors}
>
	{#if rows.length === 0}
		<p class="leftover-empty" data-testid="leftover-empty">Nessun residuo</p>
	{:else}
		<table class="leftover-table" data-testid="leftover-table">
			<thead>
				<tr>
					<th scope="col">{leftovers.columns.id}</th>
					<th scope="col">{leftovers.columns.value}</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as id (id)}
					<tr>
						<td class="leftover-id">
							<code>{id}</code>
						</td>
						<td class="leftover-value">
							{#if leftovers.valueType === 'progress'}
								<select
									value={displayById[id] ?? ''}
									onchange={(event) => {
										const raw = event.currentTarget.value;
										setProgressCell(id, isPalletProgress(raw) ? raw : undefined);
									}}
								>
									<option value="">—</option>
									<option value="partial">Parziale</option>
									<option value="done">Completo</option>
								</select>
							{:else}
								<input
									type="number"
									step="any"
									value={cellNumber(id)}
									oninput={(event) => {
										setNumberCell(id, event.currentTarget.value);
									}}
								/>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</Template>

<style>
	.leftover-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	.leftover-table th,
	.leftover-table td {
		border-bottom: 1px solid #e4e4e7;
		padding: 0.5rem 0.25rem;
		text-align: left;
	}

	.leftover-table th {
		font-weight: 500;
		color: #52525b;
	}

	.leftover-id code {
		font-size: 0.8rem;
	}

	.leftover-value input[type='number'],
	.leftover-value select {
		width: 100%;
		max-width: 10rem;
		border: 1px solid #d4d4d8;
		border-radius: 0.25rem;
		padding: 0.25rem 0.5rem;
	}

	.leftover-empty {
		margin: 0.25rem 0 0.75rem;
		color: #71717a;
		font-size: 0.875rem;
	}
</style>
