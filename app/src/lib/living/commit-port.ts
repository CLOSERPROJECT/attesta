import type { Domain } from './types.js';

/** Org Domain Commit — Task path + Job instance; no run id. */
export type CommitRequest = {
	taskPath: string;
	jobInstanceId: string;
	value: Record<string, unknown>;
};

export type CommitResult = {
	org: string;
	jobInstanceId: string;
	domain: Domain;
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
