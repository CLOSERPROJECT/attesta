/** Post-stay confirmation surface (ADR-0006 stay; no under-form Continue). */
export type StayConfirmSurface = 'once-recap' | 'many-recap-tab';

/** Stay Commit → confirmation surface + Form / timeline freshness. */
export type StayAftermath = {
	confirmSurface: StayConfirmSurface;
	/**
	 * When true, Domain may have changed: refetch form bundles for the board
	 * scope and remount Form on a cleared draft
	 * (BoardFormCache.refreshAllAfterDomainCommit).
	 * Pre-submit Form↔Recap toggles do not run this path. Once stay does not
	 * refresh (Form stays behind once-recap).
	 */
	refreshForms: boolean;
	/**
	 * Domain Commit always can unlock / lock other Tasks: refetch the board
	 * TimelineView (BoardSession.reload). Navigate follow-through already
	 * reloads; stay must too or orgActionable / step projection stays stale.
	 */
	reloadTimeline: boolean;
};

/** Stay Commit → which embed confirmation to land on. */
export function stayConfirmSurface(submitMany: boolean): StayConfirmSurface {
	return submitMany ? 'many-recap-tab' : 'once-recap';
}

/** Stay Commit → confirmation + Form / timeline freshness after Domain Commit. */
export function stayAftermath(submitMany: boolean): StayAftermath {
	return {
		confirmSurface: stayConfirmSurface(submitMany),
		refreshForms: submitMany,
		reloadTimeline: true
	};
}
