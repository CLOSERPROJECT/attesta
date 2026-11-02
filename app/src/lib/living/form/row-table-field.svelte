<script lang="ts">
	import {
		getChildPath,
		getComponent,
		getFieldErrors,
		getFieldsErrors,
		getFormContext,
		type ComponentProps
	} from '@sjsf/form';
	import type { SchemaObjectValue } from '@sjsf/form/core';
	import { omitUntouchedProgressCells } from './form-progress.js';
	import { getRowContext } from './row-context.js';

	let {
		config,
		value = $bindable(),
		uiOption
	}: ComponentProps['rowTableField'] = $props();

	const ctx = getFormContext();
	const rowCtx = getRowContext();

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const rows = $derived(rowCtx.ids);
	const progressEnums = $derived(rowCtx.progressEnums);
	// Ajv nests under map/<id>; exact getFieldErrors(map) alone is silent.
	const errors = $derived.by(() => {
		const own = getFieldErrors(ctx, config.path);
		const childPaths = rows.map((id) => getChildPath(ctx, config.path, id));
		const nested = getFieldsErrors(ctx, childPaths);
		return nested.length === 0 ? own : [...own, ...nested];
	});

	// Display SoT is sjsf value (host seeds Draft from domain at open).
	// null = clear write; missing/non-enum → undefined (available).
	const displayById = $derived.by(() => {
		const out: Record<string, string | undefined> = {};
		const allowed = new Set(progressEnums);
		for (const id of rows) {
			const cell = value?.[id];
			if (cell === null) out[id] = undefined;
			else if (typeof cell === 'string' && allowed.has(cell)) out[id] = cell;
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

	function setProgressCell(id: string, nextProgress: string | undefined): void {
		const next: SchemaObjectValue = { ...(value ?? {}) };
		// `null` = clear intent for submit (unset domain progress); omit = untouched.
		if (nextProgress === undefined) next[id] = null;
		else next[id] = nextProgress;
		commitFieldValue(omitUntouchedProgressCells(next) as SchemaObjectValue);
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
		<p class="row-empty" data-testid="row-empty">No rows</p>
	{:else}
		<table class="row-table" data-testid="row-table">
			<thead>
				<tr>
					<th scope="col">{rowCtx.columns.id}</th>
					<th scope="col">{rowCtx.columns.value}</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as id (id)}
					<tr>
						<td class="row-id">
							<code>{id}</code>
						</td>
						<td class="row-value">
							{#if rowCtx.valueType === 'progress'}
								<select
									value={displayById[id] ?? ''}
									onchange={(event) => {
										const raw = event.currentTarget.value;
										setProgressCell(
											id,
											progressEnums.includes(raw) ? raw : undefined
										);
									}}
								>
									<option value="">—</option>
									{#each progressEnums as opt (opt)}
										<option value={opt}>{opt}</option>
									{/each}
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
	.row-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	.row-table th,
	.row-table td {
		border-bottom: 1px solid #e4e4e7;
		padding: 0.5rem 0.25rem;
		text-align: left;
	}

	.row-table th {
		font-weight: 500;
		color: #52525b;
	}

	.row-id code {
		font-size: 0.8rem;
	}

	.row-value input[type='number'],
	.row-value select {
		width: 100%;
		max-width: 10rem;
		border: 1px solid #d4d4d8;
		border-radius: 0.25rem;
		padding: 0.25rem 0.5rem;
	}

	.row-empty {
		margin: 0.25rem 0 0.75rem;
		color: #71717a;
		font-size: 0.875rem;
	}
</style>
