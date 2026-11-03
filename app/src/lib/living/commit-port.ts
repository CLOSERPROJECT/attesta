import type { TaskFormBundle, Worklist } from './types.js';

/** Commit wire mint knob (omit → none on server). */
export type CommitMint =
	| { kind: 'none' }
	| { kind: 'startingNew' };

export type CommitRequest = {
	taskPath: string;
	/** Empty only with mint.startingNew; never send `_new` on the wire. */
	jobInstanceId: string;
	value: Record<string, unknown>;
	mint?: CommitMint;
};

/** Post-Commit HttpPort link-strip item (server-resolved onSubmit.http). */
export type HttpLinkItem = {
	label: string;
	url: string;
	action: 'link' | 'download';
	filename?: string;
};

/** ADR-0006 post-Commit destination (server-evaluated; SPA navigates only). */
export type CommitNext = 'task' | 'overview' | 'home';

export type CommitResult = {
	org: string;
	jobInstanceId: string;
	/** Server-projected board after this Commit (same projector as Load). */
	worklist: Worklist;
	/** Explicit destination: task | overview | home. */
	next: CommitNext;
	/** Present only when next is task (auto-advance bundle). */
	nextTaskFormBundle?: TaskFormBundle | null;
	/** Successful HttpPort side-effects; empty when Task has no onSubmit.http. */
	links?: HttpLinkItem[];
	/** HttpPort failures; Domain remains committed server-side. */
	httpErrors?: string[];
};

export type CommitOutcome = {
	jobInstanceId: string;
	worklist: Worklist;
	next: CommitNext;
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
