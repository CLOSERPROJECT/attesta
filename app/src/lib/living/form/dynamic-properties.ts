/**
 * Field-keyed dynamic map config (`dynamicProperties`) — parse + validate at load.
 * Row ids come from Domain reads; full vs partial Commit is schema.required projection.
 */

export type DynamicPropertyDef = {
	read: string;
};

export type DynamicProperties = Record<string, DynamicPropertyDef>;

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/** mapTable (cell map or object-row composition) needs dynamicProperties. */
const DYNAMIC_TABLE_FIELDS = new Set(['mapTable']);

function dynamicTableProperties(uiSchema: unknown): string[] {
	if (!isPlainObject(uiSchema)) return [];
	const found: string[] = [];
	if (typeof uiSchema['ui:field'] === 'string' && DYNAMIC_TABLE_FIELDS.has(uiSchema['ui:field'])) {
		found.push('');
	}
	for (const [key, child] of Object.entries(uiSchema)) {
		if (key.startsWith('ui:')) continue;
		if (
			isPlainObject(child) &&
			typeof child['ui:field'] === 'string' &&
			DYNAMIC_TABLE_FIELDS.has(child['ui:field'])
		) {
			found.push(key);
		}
	}
	return found;
}

export function isMapTableUiField(uiSchema: unknown, property: string): boolean {
	if (!isPlainObject(uiSchema)) return false;
	const propUi = uiSchema[property];
	return isPlainObject(propUi) && propUi['ui:field'] === 'mapTable';
}

function parseDynamicPropertyDef(
	path: string,
	property: string,
	raw: unknown
): DynamicPropertyDef {
	if (!isPlainObject(raw)) {
		throw new Error(`${path}.dynamicProperties.${property} must be a mapping`);
	}
	if (typeof raw.read !== 'string' || raw.read.length === 0) {
		throw new Error(`${path}.dynamicProperties.${property}.read must be a non-empty string`);
	}
	if (raw.completeness !== undefined) {
		throw new Error(
			`${path}.dynamicProperties.${property}.completeness is removed; use schema.required for full-map`
		);
	}
	return { read: raw.read };
}

/**
 * Validate mapTable paint vs `dynamicProperties`; reject legacy `propertyNamesFrom`.
 * Returns a shallow copy with normalized `dynamicProperties` when present.
 */
export function parseAuthoredPayload(path: string, raw: unknown): Record<string, unknown> {
	if (!isPlainObject(raw)) {
		throw new Error(`${path} must be a mapping`);
	}
	if (raw.propertyNamesFrom !== undefined) {
		throw new Error(
			`${path}: propertyNamesFrom is removed; use field-keyed dynamicProperties`
		);
	}
	if (raw.unlockFrom !== undefined) {
		if (typeof raw.unlockFrom !== 'string' || raw.unlockFrom.length === 0) {
			throw new Error(`${path}.unlockFrom must be a non-empty string task id when present`);
		}
	}
	if (raw.roles !== undefined) {
		if (!Array.isArray(raw.roles) || raw.roles.length === 0) {
			throw new Error(`${path}.roles must be a non-empty array of strings when present`);
		}
		for (let i = 0; i < raw.roles.length; i++) {
			const role = raw.roles[i];
			if (typeof role !== 'string' || role.length === 0) {
				throw new Error(`${path}.roles[${i}] must be a non-empty string`);
			}
		}
	}

	const tableProps = dynamicTableProperties(raw.uiSchema);
	const hasDynamic = raw.dynamicProperties !== undefined;

	if (tableProps.includes('')) {
		throw new Error(
			`${path}: mapTable must be under a named schema property (not root uiSchema)`
		);
	}

	if (!hasDynamic) {
		if (tableProps.length > 0) {
			throw new Error(
				`${path}: mapTable requires dynamicProperties entry for each painted property`
			);
		}
		return { ...raw };
	}

	if (!isPlainObject(raw.dynamicProperties)) {
		throw new Error(`${path}.dynamicProperties must be a mapping`);
	}

	const reads = isPlainObject(raw.reads) ? raw.reads : null;
	const schema = isPlainObject(raw.schema) ? raw.schema : null;
	const schemaProps = isPlainObject(schema?.properties) ? schema.properties : null;

	const dynamicProperties: DynamicProperties = {};
	for (const [property, defRaw] of Object.entries(raw.dynamicProperties)) {
		const def = parseDynamicPropertyDef(path, property, defRaw);
		if (!schemaProps || !(property in schemaProps)) {
			throw new Error(
				`${path}.dynamicProperties.${property}: schema.properties.${property} is required`
			);
		}
		if (!reads || !(def.read in reads)) {
			throw new Error(
				`${path}.dynamicProperties.${property}.read "${def.read}" is not in ${path}.reads`
			);
		}
		dynamicProperties[property] = def;
	}

	for (const property of tableProps) {
		if (!(property in dynamicProperties)) {
			throw new Error(
				`${path}: mapTable on "${property}" requires dynamicProperties.${property}`
			);
		}
	}

	for (const property of Object.keys(dynamicProperties)) {
		if (!tableProps.includes(property)) {
			throw new Error(
				`${path}.dynamicProperties.${property}: uiSchema.${property}.ui:field must be mapTable`
			);
		}
	}

	return { ...raw, dynamicProperties };
}

export function dynamicPropertyKeys(formConfig: unknown): string[] {
	if (!isPlainObject(formConfig)) return [];
	const map = formConfig.dynamicProperties;
	if (!isPlainObject(map)) return [];
	return Object.keys(map);
}

export function primaryDynamicProperty(formConfig: unknown): string | null {
	const keys = dynamicPropertyKeys(formConfig);
	return keys[0] ?? null;
}

export function dynamicPropertyDef(
	formConfig: unknown,
	property: string
): DynamicPropertyDef | null {
	if (!isPlainObject(formConfig) || !isPlainObject(formConfig.dynamicProperties)) {
		return null;
	}
	const raw = formConfig.dynamicProperties[property];
	if (!isPlainObject(raw)) return null;
	if (typeof raw.read !== 'string' || raw.read.length === 0) return null;
	return { read: raw.read };
}
