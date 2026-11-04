import type { HttpLinkItem } from './commit-port.js';

export type TaskRecapRequest = {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	/** Nested leaf under a Task group. */
	groupId?: string;
	groupInstanceId?: string;
};

/** Thin done-Task overview expand payload (no full schema / Domain). */
export type TaskRecap = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	groupId?: string;
	groupInstanceId?: string;
	timestamp: string;
	userId: string;
	facts: Record<string, unknown>;
	/** Authored schema property titles keyed by fact key; missing → use key. */
	titles?: Record<string, string>;
	links: HttpLinkItem[];
};

export type TaskRecapPort = {
	fetch(req: TaskRecapRequest): Promise<TaskRecap>;
};

export class TaskRecapError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'TaskRecapError';
		this.status = status;
	}
}
