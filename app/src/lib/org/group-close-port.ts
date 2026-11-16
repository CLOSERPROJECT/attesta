import type { CommitResult } from './commit-port.js';

export type CloseGroupRequest = {
	jobId: string;
	jobInstanceId: string;
	groupId: string;
};

export type CloseGroupPort = {
	close(req: CloseGroupRequest): Promise<CommitResult>;
};
