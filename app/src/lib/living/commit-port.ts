import type { TaskFormBundle, Worklist } from './types.js';

/** Commit wire mint knob (omit → none on server). */
export type CommitMint =
	| { kind: 'none' }
	| { kind: 'startingNew' };

/** Org Domain Commit — Task path + Job instance; no run id. */
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

export type CommitResult = {
	org: string;
	jobInstanceId: string;
	/** Server-projected board after this Commit (same projector as Load). */
	worklist: Worklist;
	/**
	 * ADR-0006 auto-advance: next role-eligible available Task on this instance.
	 * Omitted on handoff / finished instance — Worklist alone drives overview/home.
	 */
	nextTaskFormBundle?: TaskFormBundle | null;
	/**
	 * Interim bulk Available mounts (pre-05). Prefer nextTaskFormBundle +
	 * addressed GET /api/living/task-form for post-Commit routing.
	 */
	taskFormBundles: TaskFormBundle[];
	/** Successful HttpPort side-effects; empty when Task has no onSubmit.http. */
	links?: HttpLinkItem[];
	/** HttpPort failures; Domain remains committed server-side. */
	httpErrors?: string[];
};

/** SPA-facing Commit success for post-Commit routing (ticket 05). */
export type CommitOutcome = {
	jobInstanceId: string;
	worklist: Worklist;
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
