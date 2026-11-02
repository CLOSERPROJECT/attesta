import type { TaskFormBundle } from './form/task-form-bundle.js';
import { TaskFormError, type TaskFormPort, type TaskFormRequest } from './task-form-port.js';

export type HttpTaskFormPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

function taskFormUrl(base: string, req: TaskFormRequest): string {
	const params = new URLSearchParams({
		jobId: req.jobId,
		taskId: req.taskId
	});
	if (req.jobInstanceId !== undefined) {
		params.set('jobInstanceId', req.jobInstanceId);
	}
	return `${base}/api/living/task-form?${params}`;
}

/** Browser Adapter: GET /api/living/task-form → one Task form bundle. */
export function createHttpTaskFormPort(
	options: HttpTaskFormPortOptions = {}
): TaskFormPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async fetch(req) {
			const res = await fetchFn(taskFormUrl(base, req), {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new TaskFormError(
					detail || `living task-form failed (${res.status})`,
					res.status
				);
			}
			return (await res.json()) as TaskFormBundle;
		}
	};
}
