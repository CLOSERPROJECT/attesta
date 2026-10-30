import type { Domain } from './types.js';

/** Org Domain Commit — Task path + Job instance; no run id. */
export type CommitRequest = {
	taskPath: string;
	jobInstanceId: string;
	value: Record<string, unknown>;
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
	domain: Domain;
	/** Successful HttpPort side-effects; empty when Task has no onSubmit.http. */
	links?: HttpLinkItem[];
	/** HttpPort failures; Domain remains committed. */
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
