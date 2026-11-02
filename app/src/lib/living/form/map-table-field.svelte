<script lang="ts" module>
	declare module '@sjsf/form' {
		interface UiOptions {
			columns?: { id?: string; value?: string };
		}
	}
</script>

<script lang="ts">
	import {
		getChildPath,
		getComponent,
		getFieldComponent,
		getFieldErrors,
		getFormContext,
		getStableConfig,
		retrieveSchema,
		retrieveTranslate,
		retrieveUiOption,
		retrieveUiSchema,
		uiTitleOption,
		type ComponentProps,
		type Config,
		type Schema,
		type UiSchema,
		type UiSchemaDefinition
	} from '@sjsf/form';
	import type { SchemaObjectValue, SchemaValue } from '@sjsf/form/core';
	import {
		createObjectContext,
		setObjectContext
	} from '@sjsf/form/fields/object/context.svelte';
	import {
		objectRowMapColumnKeys,
		writeObjectRowMapCell
	} from './object-row-weigh.js';
	import { readColumns } from './read-columns.js';

	let {
		config,
		value = $bindable(),
		uiOption,
		translate
	}: ComponentProps['mapTableField'] = $props();

	const ctx = getFormContext();

	const objCtx = createObjectContext({
		ctx,
		config: () => config,
		value: () => value,
		setValue: (v: SchemaObjectValue | null | undefined) => {
			value = v;
		},
		get translate() {
			return translate;
		}
	});
	setObjectContext(objCtx);

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));

	const columns = $derived(readColumns({ 'ui:options': { columns: uiOption('columns') } }));

	function isObjectRowSchema(schema: Schema): boolean {
		if (schema.properties && typeof schema.properties === 'object') return true;
		const t = schema.type;
		if (t === 'object') return true;
		if (Array.isArray(t) && t.includes('object')) return true;
		return false;
	}

	/** Object map → property columns; cell map → single value column. */
	const objectMap = $derived.by(() => {
		const order = objCtx.propertiesOrder();
		const sample = order[0];
		if (!sample) return false;
		const rowCfg = objCtx.propertyConfig(config, sample, objCtx.isAdditionalProperty(sample));
		return isObjectRowSchema(rowCfg.schema);
	});

	const columnKeys = $derived.by(() => {
		if (!objectMap) return [] as string[];
		const order = objCtx.propertiesOrder();
		const sample = order[0];
		if (!sample) return [] as string[];
		const rowCfg = objCtx.propertyConfig(config, sample, objCtx.isAdditionalProperty(sample));
		return objectRowMapColumnKeys(rowCfg.schema.properties);
	});

	const valueColumnTitle = $derived.by(() => {
		if (columns.value !== 'Value') return columns.value;
		const order = objCtx.propertiesOrder();
		const sample = order[0];
		if (!sample) return columns.value;
		const rowCfg = objCtx.propertyConfig(config, sample, objCtx.isAdditionalProperty(sample));
		return rowCfg.schema.title ?? columns.value;
	});

	function withHiddenTitle(uiSchema: UiSchema): UiSchema {
		const options =
			uiSchema['ui:options'] && typeof uiSchema['ui:options'] === 'object'
				? { ...(uiSchema['ui:options'] as Record<string, unknown>) }
				: {};
		return {
			...uiSchema,
			'ui:options': {
				...options,
				hideTitle: true
			}
		};
	}

	function asUiSchemaDefinition(raw: unknown): UiSchemaDefinition | undefined {
		if (raw === undefined || raw === null || typeof raw === 'boolean') return undefined;
		if (typeof raw !== 'object' || Array.isArray(raw)) return undefined;
		return raw as UiSchemaDefinition;
	}

	function cellConfig(rowCfg: Config, column: string): Config {
		const rowValue = (rowCfg.value?.() ?? undefined) as SchemaObjectValue | undefined;
		const definition = rowCfg.schema.properties?.[column] ?? false;
		const schema =
			typeof definition === 'boolean'
				? ({} as Schema)
				: retrieveSchema(ctx, definition, rowValue?.[column]);
		const uiSchema = withHiddenTitle(
			retrieveUiSchema(ctx, asUiSchemaDefinition(rowCfg.uiSchema[column]))
		);
		const required = new Set(rowCfg.schema.required ?? []).has(column);
		return getStableConfig(ctx, {
			path: getChildPath(ctx, rowCfg.path, column),
			title: uiTitleOption(ctx, uiSchema) ?? schema.title ?? column,
			schema,
			uiSchema,
			required,
			value: () => rowValue?.[column]
		});
	}

	/** Cell-map: additionalProperties schema is the cell itself. */
	function scalarCellConfig(rowCfg: Config): Config {
		return getStableConfig(ctx, {
			path: rowCfg.path,
			title: rowCfg.title,
			schema: rowCfg.schema,
			uiSchema: withHiddenTitle(rowCfg.uiSchema),
			required: rowCfg.required,
			value: rowCfg.value
		});
	}

	function setCell(rowKey: string, column: string, cell: SchemaValue | undefined): void {
		const row = {
			...((value?.[rowKey] as SchemaObjectValue | undefined) ?? {})
		} as Record<string, unknown>;
		objCtx.set(rowKey, writeObjectRowMapCell(row, column, cell) as SchemaObjectValue | undefined);
	}

	function setScalarCell(rowKey: string, cell: SchemaValue | undefined): void {
		objCtx.set(rowKey, cell as SchemaObjectValue | undefined);
	}

	function columnHeaderTitle(column: string): string {
		const order = objCtx.propertiesOrder();
		const sample = order[0];
		if (!sample) return column;
		const rowCfg = objCtx.propertyConfig(config, sample, objCtx.isAdditionalProperty(sample));
		return cellConfig(rowCfg, column).title;
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
	<table class="map-table" data-testid="map-table">
		<thead>
			<tr>
				<th scope="col">{columns.id}</th>
				{#if objectMap}
					{#each columnKeys as col (col)}
						<th scope="col">{columnHeaderTitle(col)}</th>
					{/each}
				{:else}
					<th scope="col">{valueColumnTitle}</th>
				{/if}
			</tr>
		</thead>
		<tbody>
			{#each objCtx.propertiesOrder() as rowKey (objCtx.key(rowKey))}
				{@const isAdditional = objCtx.isAdditionalProperty(rowKey)}
				{@const rowCfg = objCtx.propertyConfig(config, rowKey, isAdditional)}
				<tr data-testid="map-row" data-row={rowKey}>
					<td><code>{rowKey}</code></td>
					{#if objectMap}
						{#each columnKeys as col (col)}
							{@const cellCfg = cellConfig(rowCfg, col)}
							{@const Field = getFieldComponent(ctx, cellCfg)}
							<td data-testid="map-cell" data-col={col}>
								<Field
									type="field"
									bind:value={
										() =>
											(value?.[rowKey] as SchemaObjectValue | undefined)?.[col] as undefined,
										(v) => setCell(rowKey, col, v)
									}
									config={cellCfg}
									uiOption={(opt) => retrieveUiOption(ctx, cellCfg, opt)}
									translate={retrieveTranslate(ctx, cellCfg)}
								/>
							</td>
						{/each}
					{:else}
						{@const cellCfg = scalarCellConfig(rowCfg)}
						{@const Field = getFieldComponent(ctx, cellCfg)}
						<td data-testid="map-cell" data-col="value">
							<Field
								type="field"
								bind:value={
									() => value?.[rowKey] as undefined,
									(v) => setScalarCell(rowKey, v)
								}
								config={cellCfg}
								uiOption={(opt) => retrieveUiOption(ctx, cellCfg, opt)}
								translate={retrieveTranslate(ctx, cellCfg)}
							/>
						</td>
					{/if}
				</tr>
			{/each}
		</tbody>
	</table>
</Template>

<style>
	.map-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}
	.map-table th,
	.map-table td {
		border-bottom: 1px solid #e4e4e7;
		padding: 0.35rem 0.25rem;
		text-align: left;
		vertical-align: top;
	}
</style>
