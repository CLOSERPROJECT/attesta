/** Post-stay confirmation surface (ADR-0006 stay; no under-form Continue / HttpLinkStrip). */
export type StayConfirmSurface = 'once-recap' | 'many-recap';

/** Stay Commit → confirmation surface + Form freshness policy. */
export type StayAftermath = {
	confirmSurface: StayConfirmSurface;
	/**
	 * When true, Domain may have changed: refetch form bundles for the board
	 * scope and remount Form on a cleared draft
	 * (BoardSession.domainChanged: follow 'stay' with submitMany).
	 * Pre-submit composer open/close does not run this path. Once stay does not
	 * refresh (Form stays behind once-recap).
	 * Stay still reloads the timeline and hydrates the Worklist via domainChanged.
	 */
	refreshForms: boolean;
	/**
	 * Drop session under-form leave links. Stay confirmation owns links on the
	 * once/many recap tables; keeping session links would bleed into any other
	 * open TaskForm (e.g. lot_processing/startLot after intake/register).
	 */
	clearUnderFormLinks: boolean;
};

/** Stay Commit → which embed confirmation to land on. */
export function stayConfirmSurface(submitMany: boolean): StayConfirmSurface {
	return submitMany ? 'many-recap' : 'once-recap';
}

/** Stay Commit → confirmation + Form freshness after Domain Commit. */
export function stayAftermath(submitMany: boolean): StayAftermath {
	return {
		confirmSurface: stayConfirmSurface(submitMany),
		refreshForms: submitMany,
		clearUnderFormLinks: true
	};
}
