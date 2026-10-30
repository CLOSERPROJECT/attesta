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
	/** Replaces prior Load/Commit bundle set (available + provisional). */
	taskFormBundles: TaskFormBundle[];
	/** Successful HttpPort side-effects; empty when Task has no onSubmit.http. */
	links?: HttpLinkItem[];
	/** HttpPort failures; Domain remains committed server-side. */
	httpErrors?: string[];
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
