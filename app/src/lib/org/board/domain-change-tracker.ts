/**
 * Decides which Domain revisions came from someone else. Relies on the server
 * contract that each successful write bumps the revision by exactly 1, so an own
 * Commit can be told apart from foreign writes whatever order the events stream
 * and the Commit response arrive in.
 */
export type DomainChangeTracker = {
	/** The board loaded data at `revision`: everything up to it is on screen. */
	reset(revision: number): void;
	commitStarted(): void;
	commitSettled(revision: number | null): void;
	/** Revision from the events stream, including the initial event of each connection. */
	remote(revision: number): void;
	/** Highest revision accounted for; null before the first reset. */
	readonly known: number | null;
};

export function createDomainChangeTracker(
	onForeign: (revision: number) => void
): DomainChangeTracker {
	let known: number | null = null;
	let inFlight = false;
	let held: number[] = [];

	return {
		reset(revision) {
			known = revision;
			held = [];
		},
		commitStarted() {
			inFlight = true;
		},
		commitSettled(revision) {
			if (known === null) {
				inFlight = false;
				held = [];
				return;
			}
			const base = known;
			if (revision !== null) {
				const foreign = held.filter((h) => h !== revision && h > base);
				const gap = revision > base + 1;
				const next = Math.max(base, revision, ...held);
				known = next;
				held = [];
				inFlight = false;
				if (foreign.length > 0 || gap) onForeign(next);
				return;
			}
			const foreign = held.filter((h) => h > base);
			known = Math.max(base, ...held);
			held = [];
			inFlight = false;
			if (foreign.length > 0) onForeign(known);
		},
		remote(revision) {
			if (known === null || revision <= known) return;
			if (inFlight) {
				held.push(revision);
				return;
			}
			known = revision;
			onForeign(revision);
		},
		get known() {
			return known;
		}
	};
}
