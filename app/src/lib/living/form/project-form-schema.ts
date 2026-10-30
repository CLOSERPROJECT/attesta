import type { Schema, UiSchemaRoot } from '@sjsf/form';
import type { DomainSnapshot } from './domain-snapshot.js';
import {
	dynamicPropertyDef,
	dynamicPropertyKeys,
	primaryDynamicProperty
} from './dynamic-properties.js';
import { paintAuthoredFieldMarkers } from './authored-field-registry.js';
import {
	formatResolveReadsError,
	parseReadDef,
	resolveRead,
	type ResolveReadOptions,
	type ResolveReadsError
} from './resolve-reads.js';

export type ProjectFormSchemaError =
	| ResolveReadsError
	| { kind: 'invalid-form'; message: string };

export type ProjectedForm = {
	schema: Schema;
	uiSchema: UiSchemaRoot;
	leftoverIds: string[];
	columns: { id: string; value: string };
	primaryDynamicProperty: string | null;
};

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function asRecord(value: unknown): Record<string, unknown> | null {
	return isPlainObject(value) ? value : null;
}

/** `ui:options.columns` labels (leftover / map-table Id header); defaults match Italian chrome. */
export function readColumns(uiSchema: Record<string, unknown>): { id: string; value: string } {
	const options = asRecord(uiSchema['ui:options']);
	const columns = options ? asRecord(options.columns) : null;
	return {
		id: typeof columns?.id === 'string' ? columns.id : 'Id',
		value: typeof columns?.value === 'string' ? columns.value : 'Valore'
	};
}

/**
 * Inject `enum` on a root string property, or on matching `items.properties` under
 * every array property (fallback for nested enumReads).
 */
function injectEnumOnSchemaProperty(
	schemaProperties: Record<string, unknown>,
	property: string,
	ids: string[]
): boolean {
	const propSchema = asRecord(schemaProperties[property]);
	if (propSchema) {
		schemaProperties[property] = { ...propSchema, enum: ids };
		return true;
	}

	let found = false;
	for (const [key, raw] of Object.entries(schemaProperties)) {
		const arr = asRecord(raw);
		if (!arr || arr.type !== 'array') continue;
		const items = asRecord(arr.items);
		if (!items) continue;
		const itemProps = asRecord(items.properties);
		if (!itemProps) continue;
		const nested = asRecord(itemProps[property]);
		if (!nested) continue;
		schemaProperties[key] = {
			...arr,
			items: {
				...items,
				properties: {
					...itemProps,
					[property]: { ...nested, enum: ids }
				}
			}
		};
		found = true;
	}
	return found;
}

/**
 * Inject `propertyNames.enum` from `dynamicProperties` + `reads`, string-field
 * `enum` from `enumReads` + `reads`, and map authored `ui:field` markers → sjsf
 * `ui:components` via authored-field-registry.
 * Completeness all|some is parsed at load; submit/unlock differences follow that mode.
 */
export async function projectFormSchema(
	snapshot: DomainSnapshot,
	formConfig: unknown,
	readOptions: ResolveReadOptions = {}
): Promise<{ ok: true; value: ProjectedForm } | { ok: false; error: ProjectFormSchemaError }> {
	if (!isPlainObject(formConfig)) {
		return {
			ok: false,
			error: { kind: 'invalid-form', message: 'form config must be a mapping' }
		};
	}
	if (!isPlainObject(formConfig.schema)) {
		return {
			ok: false,
			error: { kind: 'invalid-form', message: 'form.schema must be a mapping' }
		};
	}

	const uiSchemaRaw = asRecord(formConfig.uiSchema) ?? {};
	const schemaRoot = { ...(formConfig.schema as Record<string, unknown>) };
	const schemaProperties = asRecord(schemaRoot.properties)
		? { ...(schemaRoot.properties as Record<string, unknown>) }
		: {};

	const propertyKeys = dynamicPropertyKeys(formConfig);
	let leftoverIds: string[] = [];
	let columns = { id: 'Id', value: 'Valore' };
	const primaryKey = primaryDynamicProperty(formConfig);
	const uiSchema: Record<string, unknown> = { ...uiSchemaRaw };
	let mutatedProperties = false;

	for (const property of propertyKeys) {
		const def = dynamicPropertyDef(formConfig, property);
		if (!def) {
			return {
				ok: false,
				error: {
					kind: 'invalid-form',
					message: `dynamicProperties.${property} is invalid`
				}
			};
		}
		const reads = asRecord(formConfig.reads);
		if (!reads) {
			return {
				ok: false,
				error: {
					kind: 'invalid-form',
					message: 'dynamicProperties requires reads'
				}
			};
		}
		const readRaw = reads[def.read];
		const parsed = parseReadDef(readRaw);
		if (!parsed.ok) return parsed;

		const resolved = await resolveRead(snapshot, parsed.value, readOptions);
		if (!resolved.ok) return resolved;

		const propSchema = asRecord(schemaProperties[property]);
		if (!propSchema) {
			return {
				ok: false,
				error: {
					kind: 'invalid-form',
					message: `dynamicProperties.${property} requires schema.properties.${property}`
				}
			};
		}
		schemaProperties[property] = {
			...propSchema,
			propertyNames: { enum: resolved.ids }
		};
		mutatedProperties = true;

		const propUi = asRecord(uiSchema[property]) ?? {};
		uiSchema[property] = paintAuthoredFieldMarkers(propUi);

		// Single leftover field per Substep in current fixtures; expose that map.
		if (property === primaryKey) {
			leftoverIds = resolved.ids;
			columns = readColumns(uiSchema[property] as Record<string, unknown>);
		}
	}

	// String-field selects: inject `enum` from named reads (root or array-item properties).
	const enumReads = asRecord(formConfig.enumReads);
	if (enumReads) {
		const reads = asRecord(formConfig.reads);
		if (!reads) {
			return {
				ok: false,
				error: {
					kind: 'invalid-form',
					message: 'enumReads requires reads'
				}
			};
		}
		for (const [property, readNameRaw] of Object.entries(enumReads)) {
			if (typeof readNameRaw !== 'string' || readNameRaw.length === 0) {
				return {
					ok: false,
					error: {
						kind: 'invalid-form',
						message: `enumReads.${property} must be a non-empty read name`
					}
				};
			}
			const parsed = parseReadDef(reads[readNameRaw]);
			if (!parsed.ok) return parsed;
			const resolved = await resolveRead(snapshot, parsed.value, readOptions);
			if (!resolved.ok) return resolved;

			const injected = injectEnumOnSchemaProperty(
				schemaProperties,
				property,
				resolved.ids
			);
			if (!injected) {
				return {
					ok: false,
					error: {
						kind: 'invalid-form',
						message: `enumReads.${property} requires schema.properties.${property} or array items.properties.${property}`
					}
				};
			}
			mutatedProperties = true;
		}
	}

	if (mutatedProperties) {
		schemaRoot.properties = schemaProperties;
	}

	const rootOptions = asRecord(uiSchema['ui:options']) ?? {};
	uiSchema['ui:options'] = {
		...rootOptions,
		expandable: false
	};

	return {
		ok: true,
		value: {
			schema: schemaRoot as Schema,
			uiSchema: uiSchema as UiSchemaRoot,
			leftoverIds,
			columns,
			primaryDynamicProperty: primaryKey
		}
	};
}

export function formatProjectFormSchemaError(error: ProjectFormSchemaError): string {
	if (error.kind === 'invalid-form') return `${error.kind}: ${error.message}`;
	return formatResolveReadsError(error);
}

export function leftoversIdentity(ids: string[]): string {
	return [...ids].sort().join('|');
}
