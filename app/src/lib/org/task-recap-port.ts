import type { HttpLinkItem } from './commit-port.js';
import type { StreamScope } from './stream-scope.js';

export type TaskRecapRequest = StreamScope & {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
};

export type TaskRecapCol = {
	key: string;
	title: string;
};

export type TaskRecapRow = {
	key: string;
	cells: Record<string, unknown>;
	/** Entity mode only: minted → Created, patched → Updated. */
	touch?: 'minted' | 'patched';
	submissionId: string;
	timestamp: string;
	userId: string;
	userName: string;
	links?: HttpLinkItem[];
};

/** Server-projected overview table for one Submission address. */
export type TaskRecapTable = {
	mode: 'entity' | 'submission';
	columns: TaskRecapCol[];
	rows: TaskRecapRow[];
};

export const EMPTY_TASK_RECAP_TABLE: TaskRecapTable = {
	mode: 'submission',
	columns: [],
	rows: []
};

export type TaskRecapPort = {
	fetch(req: TaskRecapRequest): Promise<TaskRecapTable>;
};

export class TaskRecapError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'TaskRecapError';
		this.status = status;
	}
}
