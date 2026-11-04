/**
 * Generic hiddenField chrome: paint Config title when ui:options.presenceMark
 * is set and the bound string is non-empty. Plant-agnostic.
 */
export function shouldPaintPresenceMark(
	presenceMark: boolean | undefined,
	value: string | undefined | null
): boolean {
	if (presenceMark !== true) return false;
	if (value == null) return false;
	return String(value).trim() !== '';
}
