import type { TaskFormBundle, Worklist } from './types.js';

/** Commit wire mint knob (omit → none on server). */
export type CommitMint =
	| { kind: 'none' }
	| { kind: 'startingNew' }
	| { kind: 'spawningNew' };

export type CommitRequest = {
	taskPath: string;
	/** Empty only with mint.startingNew; never send `_new` on the wire. */
	jobInstanceId: string;
	value: Record<string, unknown>;
	mint?: CommitMint;
	/**
	 * Client draft-bag instance key (see draftBagInstanceKey). Not on the wire.
	 * Omit → clear using jobInstanceId only.
	 */
	draftBagInstanceKey?: string;
};

/** Post-Commit HttpPort link-strip item (server-resolved onSubmit.http). */
export type HttpLinkItem = {
	label: string;
	url: string;
	action: 'link' | 'download';
	filename?: string;
};

/**
 * ADR-0006 Commit `navigation` (server-evaluated; SPA navigates only).
 * `stay` parks on the submitted form; Continue uses `continueNavigation`.
 */
export type CommitNavigation = 'stay' | 'task' | 'overview' | 'home';

/** Deferred Continue target when `navigation` is stay (never stay). */
export type ContinueNavigation = 'task' | 'overview' | 'home';

export type CommitResult = {
	org: string;
	jobInstanceId: string;
	/** Server-projected board after this Commit (same projector as Load). */
	worklist: Worklist;
	/** Explicit destination: stay | task | overview | home. */
	navigation: CommitNavigation;
	/**
	 * When navigation is stay: deferred Continue target (home | overview | task).
	 * Omitted otherwise.
	 */
	continueNavigation?: ContinueNavigation | null;
	/** Present when navigation or continueNavigation is task. */
	nextTaskFormBundle?: TaskFormBundle | null;
	/** Successful HttpPort side-effects; empty when Task has no onSubmit.http. */
	links?: HttpLinkItem[];
	/** HttpPort failures; Domain remains committed server-side. */
	httpErrors?: string[];
};

export type CommitOutcome = {
	jobInstanceId: string;
	worklist: Worklist;
	navigation: CommitNavigation;
	continueNavigation: ContinueNavigation | null;
	nextTaskFormBundle: TaskFormBundle | null;
};

export type CommitDomainPort = {
	commit(req: CommitRequest): Promise<CommitResult>;
};

export class CommitDomainError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'CommitDomainError';
		this.status = status;
	}
}
