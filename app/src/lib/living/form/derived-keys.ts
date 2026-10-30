/**
 * Derived keys for ConfigForm: concrete schema paths (dotted, or JSON Pointer
 * when a segment contains `.` — B4 `{loadId}.{i}`) plus host-side map-star
 * sugar (`weights.*.panelCount` → concrete keys + `__ROW_ID__` in expr).
 * ConfigForm and fields consume only concrete keys; Session expands authored
 * maps through this interface before construct.
 */

/** Authored / expanded derived entry — structural match for ConfigFormDerivedEntry. */
export type DerivedKeyEntry = {
	expr: string;
	/** ADR-0016: default false — operator cannot keep a dirty value. */
	overridable?: boolean;
};

const MAP_STAR = /^(.+)\.\*\.(.+)$/;

/** Join path segments; JSON Pointer when any segment contains `.` (ambiguous dotted join). */
export function encodeFormPath(segments: readonly string[]): string {
	if (segments.length === 0) return '';
	if (segments.some((s) => s.includes('.'))) {
		return (
			'/' +
			segments
				.map((s) => s.replace(/~/g, '~0').replace(/\//g, '~1'))
				.join('/')
		);
	}
	return segments.join('.');
}

export function parseFormPath(path: string): string[] {
	if (path.startsWith('/')) {
		return path
			.slice(1)
			.split('/')
			.filter((segment) => segment.length > 0)
			.map((segment) => segment.replace(/~1/g, '/').replace(/~0/g, '~'));
	}
	if (path.length === 0) return [];
	return path.split('.');
}

export function getAtPath(root: unknown, path: string): unknown {
	const segments = parseFormPath(path);
	let cur: unknown = root;
	for (const segment of segments) {
		if (cur === null || typeof cur !== 'object') return undefined;
		cur = (cur as Record<string, unknown>)[segment];
	}
	return cur;
}

/** Immutable set; creates intermediate plain objects as needed. */
export function setAtPath(
	root: Record<string, unknown>,
	path: string,
	value: unknown
): Record<string, unknown> {
	const segments = parseFormPath(path);
	if (segments.length === 0) {
		if (value === null || typeof value !== 'object' || Array.isArray(value)) {
			throw new Error('setAtPath: empty path requires an object value');
		}
		return value as Record<string, unknown>;
	}
	const out: Record<string, unknown> = { ...root };
	let cur: Record<string, unknown> = out;
	for (let i = 0; i < segments.length - 1; i++) {
		const segment = segments[i]!;
		const next = cur[segment];
		const child =
			next !== null && typeof next === 'object' && !Array.isArray(next)
				? { ...(next as Record<string, unknown>) }
				: {};
		cur[segment] = child;
		cur = child;
	}
	cur[segments[segments.length - 1]!] = value;
	return out;
}

/** Join sjsf `config.path` segments into the ConfigForm derived key (pointer if needed). */
export function configPathKey(path: readonly (string | number)[]): string {
	return encodeFormPath(path.map(String));
}

/**
 * Expand map-level derived keys `map.*.field` into concrete ConfigForm paths
 * (`map.<id>.field`, or JSON Pointer when `id` contains `.`). Authored expr may
 * embed `__ROW_ID__` (replaced per id) for `$lookup($data.weights, "__ROW_ID__")`.
 */
export function expandMapDerived(
	derived: Record<string, DerivedKeyEntry>,
	rowIds: readonly string[]
): Record<string, DerivedKeyEntry> {
	const out: Record<string, DerivedKeyEntry> = {};
	for (const [key, entry] of Object.entries(derived)) {
		const match = MAP_STAR.exec(key);
		if (!match) {
			out[key] = entry;
			continue;
		}
		const mapProp = match[1]!;
		const field = match[2]!;
		for (const id of rowIds) {
			out[encodeFormPath([mapProp, id, field])] = {
				expr: entry.expr.split('__ROW_ID__').join(id),
				...(entry.overridable !== undefined ? { overridable: entry.overridable } : {})
			};
		}
	}
	return out;
}
