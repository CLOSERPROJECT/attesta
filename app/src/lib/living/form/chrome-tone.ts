/** Presentation tones for Attesta chrome (recap rows). Not kernel / derived. */

export type ChromeTone = 'info' | 'warning' | 'danger' | 'green';

export type ChromeToneBand = {
	min: number;
	tone: ChromeTone;
};

const TONES = new Set<string>(['info', 'warning', 'danger', 'green']);

export function isChromeTone(value: unknown): value is ChromeTone {
	return typeof value === 'string' && TONES.has(value);
}

/**
 * Resolve row tone: when `bands` is non-empty, first band with value ≥ min wins;
 * non-numeric values yield no band tone. Otherwise use optional static `tone`.
 */
export function resolveChromeTone(
	value: unknown,
	options: { tone?: ChromeTone; bands?: readonly ChromeToneBand[] }
): ChromeTone | null {
	const bands = options.bands;
	if (bands !== undefined && bands.length > 0) {
		if (typeof value !== 'number' || !Number.isFinite(value)) return null;
		for (const band of bands) {
			if (value >= band.min) return band.tone;
		}
		return null;
	}
	return options.tone ?? null;
}
