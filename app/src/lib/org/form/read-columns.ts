import * as m from '#lib/paraglide/messages.js';

/** `ui:options.columns` labels (map-table Id header); locale defaults when unset. */

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function readColumns(uiSchema: Record<string, unknown>): { id: string; value: string } {
	const options = isPlainObject(uiSchema['ui:options']) ? uiSchema['ui:options'] : null;
	const columns = options && isPlainObject(options.columns) ? options.columns : null;
	return {
		id: typeof columns?.id === 'string' ? columns.id : m.column_id(),
		value: typeof columns?.value === 'string' ? columns.value : m.column_value()
	};
}
