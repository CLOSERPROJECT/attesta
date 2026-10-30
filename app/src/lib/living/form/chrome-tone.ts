/** Presentation tones for Attesta chrome display rows. Not kernel / derived. */

export type ChromeTone = 'info' | 'warning' | 'danger' | 'green';

const TONES = new Set<string>(['info', 'warning', 'danger', 'green']);

export function isChromeTone(value: unknown): value is ChromeTone {
	return typeof value === 'string' && TONES.has(value);
}
