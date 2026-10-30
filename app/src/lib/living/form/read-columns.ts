/** `ui:options.columns` labels (map-table Id header); defaults match Italian chrome. */

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function readColumns(uiSchema: Record<string, unknown>): { id: string; value: string } {
	const options = isPlainObject(uiSchema['ui:options']) ? uiSchema['ui:options'] : null;
	const columns = options && isPlainObject(options.columns) ? options.columns : null;
	return {
		id: typeof columns?.id === 'string' ? columns.id : 'Id',
		value: typeof columns?.value === 'string' ? columns.value : 'Valore'
	};
}
