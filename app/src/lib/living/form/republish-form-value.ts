/**
 * Why: force a whole-object `setBoundValue` so DraftSession (`$state.raw`) sees a
 * new root. Sole production caller is `syncDeepLocalToDraft`, which passes
 * `JSON.parse` of a stringify snapshot — already a detached plain tree.
 */
export function republishBoundValue(
	getValue: () => Record<string, unknown>,
	setBoundValue: (next: Record<string, unknown>) => void
): void {
	setBoundValue(getValue());
}
