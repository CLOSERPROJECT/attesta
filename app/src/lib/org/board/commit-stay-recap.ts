/** Post-stay confirmation surface (ADR-0006 stay; no under-form Continue). */
export type StayConfirmSurface = 'once-recap' | 'many-recap-tab';

/** Stay Commit → confirmation surface + Form draft lifetime. */
export type StayAftermath = {
	confirmSurface: StayConfirmSurface;
	/**
	 * When true, Form stays mountable across Recap and must bind a new empty draft
	 * (session already cleared the bag; BoardFormCache.beginFreshDraft advances identity).
	 * Pre-submit Form↔Recap toggles do not run this path.
	 */
	freshDraft: boolean;
};

/** Stay Commit → which embed confirmation to land on. */
export function stayConfirmSurface(submitMany: boolean): StayConfirmSurface {
	return submitMany ? 'many-recap-tab' : 'once-recap';
}

/** Stay Commit → confirmation + whether Form identity must advance. */
export function stayAftermath(submitMany: boolean): StayAftermath {
	return {
		confirmSurface: stayConfirmSurface(submitMany),
		freshDraft: submitMany
	};
}
