import type { Schema } from '@sjsf/form';

/** Widget attribute bags that carry `placeholder` in @sjsf/shadcn4-theme. */
const SHADCN_PLACEHOLDER_KEYS = ['shadcn4Text', 'shadcn4Number', 'shadcn4Select'] as const;

/** Basic-theme fallbacks (same attribute shape). */
const BASIC_PLACEHOLDER_KEYS = ['text', 'number', 'select'] as const;

function asRecord(value: unknown): Record<string, unknown> | null {
	if (value === null || typeof value !== 'object' || Array.isArray(value)) {
		return null;
	}
	return value as Record<string, unknown>;
}

export function defaultPlaceholderText(fieldKey: string, schema: Schema): string {
	const title = typeof schema.title === 'string' ? schema.title.trim() : '';
	const label = title.length > 0 ? title : fieldKey;
	return `Inserisci ${label}…`;
}

function schemaTypes(schema: Schema): string[] {
	const t = schema.type;
	if (typeof t === 'string') return [t];
	if (Array.isArray(t)) return t.map(String);
	return [];
}

/** Placeholders apply to string/number inputs and selects — not booleans, objects, arrays. */
function acceptsPlaceholder(schema: Schema): boolean {
	if (schema.readOnly === true) return false;
	const types = schemaTypes(schema);
	return types.some((t) => t === 'string' || t === 'number' || t === 'integer');
}

function fieldHasPlaceholder(fieldUi: Record<string, unknown>): boolean {
	if (typeof fieldUi['ui:placeholder'] === 'string') return true;
	const opts = asRecord(fieldUi['ui:options']);
	if (!opts) return false;
	if (typeof opts.placeholder === 'string') return true;
	for (const key of [...SHADCN_PLACEHOLDER_KEYS, ...BASIC_PLACEHOLDER_KEYS]) {
		const widget = asRecord(opts[key]);
		if (widget && typeof widget.placeholder === 'string') return true;
	}
	return false;
}

/**
 * Inject default `placeholder` into shadcn4 widget ui:options when missing.
 * Skips chrome / readOnly / non-input types; never overwrites authored placeholders.
 */
export function injectDefaultPlaceholders(
	uiSchema: Record<string, unknown>,
	properties: Record<string, Schema>,
	skipIds: ReadonlySet<string>
): void {
	for (const [key, schema] of Object.entries(properties)) {
		if (skipIds.has(key)) continue;
		if (!acceptsPlaceholder(schema)) continue;

		const existing = asRecord(uiSchema[key]) ?? {};
		if (fieldHasPlaceholder(existing)) continue;

		const text = defaultPlaceholderText(key, schema);
		const existingOpts = asRecord(existing['ui:options']) ?? {};
		const nextOpts: Record<string, unknown> = { ...existingOpts };

		for (const widgetKey of SHADCN_PLACEHOLDER_KEYS) {
			const prev = asRecord(existingOpts[widgetKey]) ?? {};
			nextOpts[widgetKey] = { ...prev, placeholder: text };
		}

		uiSchema[key] = {
			...existing,
			'ui:options': nextOpts
		};
	}
}
