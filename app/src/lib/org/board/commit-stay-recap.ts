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

/**
 * Submit-many embed land surface (Option C).
 * Mode owns composer / Done-band visibility — remountForms, draftGeneration,
 * and confirmRecap catch-up must not infer open/closed.
 */
export type ManyLandSurface = 'empty-compose' | 'done-band' | 'composing';

export type ManyLandEvent =
	| { type: 'commitSuccess' }
	| { type: 'discard'; submissionCount: number }
	| { type: 'openComposer' };

/** First paint for a many embed address — not re-run on submission/recap catch-up. */
export function initialManyLand(input: {
	submissionCount: number;
	confirmRecap: boolean;
}): ManyLandSurface {
	if (input.submissionCount === 0 && !input.confirmRecap) return 'empty-compose';
	return 'done-band';
}

export function reduceManyLand(surface: ManyLandSurface, event: ManyLandEvent): ManyLandSurface {
	switch (event.type) {
		case 'commitSuccess':
			return 'done-band';
		case 'discard':
			return event.submissionCount === 0 ? 'empty-compose' : 'done-band';
		case 'openComposer':
			return 'composing';
	}
}

export function showComposer(surface: ManyLandSurface): boolean {
	return surface !== 'done-band';
}

/** Done/recap band + Add chrome (hidden only on empty-compose auto-open). */
export function showDoneBand(surface: ManyLandSurface): boolean {
	return surface !== 'empty-compose';
}
