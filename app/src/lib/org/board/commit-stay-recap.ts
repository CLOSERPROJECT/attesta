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

/** Inputs for submit-many composer open/closed (Option C). */
export type ManyComposerVisibility = {
	composerOpen: boolean;
	/**
	 * Latched true after a successful many stay Commit (or Discard). Blocks
	 * manyEmpty auto-open until Add — closes the race where remountForms
	 * (register progressive chrome) paints a fresh form before confirmRecap /
	 * submissions land.
	 */
	holdClosed: boolean;
	submissionCount: number;
	confirmRecap: boolean;
};

/**
 * Whether the submit-many embed shows the form pane.
 * Empty land (N=0, no stay cue, not held) auto-opens; after Commit with data /
 * hold, stays closed until Add sets composerOpen.
 */
export function manyShowComposer(v: ManyComposerVisibility): boolean {
	if (v.holdClosed) return false;
	const manyEmpty = v.submissionCount === 0 && !v.confirmRecap;
	return v.composerOpen || manyEmpty;
}

/** Empty land for spacing / Done-band gate (held post-Commit counts as non-empty). */
export function manyLandEmpty(v: Omit<ManyComposerVisibility, 'composerOpen'>): boolean {
	if (v.holdClosed) return false;
	return v.submissionCount === 0 && !v.confirmRecap;
}
