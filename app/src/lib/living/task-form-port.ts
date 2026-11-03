import type { TaskFormBundle } from './form/task-form-bundle.js';

export type TaskFormRequest = {
	jobId: string;
	taskId: string;
	/** Empty / omitted = provisional start; wire may also send `new`. */
	jobInstanceId?: string;
	/** Nested leaf under a Task group. */
	groupId?: string;
	/** Empty / omitted / `new` = provisional spawn. */
	groupInstanceId?: string;
};

export type TaskFormPort = {
	fetch(req: TaskFormRequest): Promise<TaskFormBundle>;
};

export class TaskFormError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'TaskFormError';
		this.status = status;
	}
}
