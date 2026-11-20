import * as m from '#lib/paraglide/messages.js';

import type { TaskStatus, TaskSubmit } from './types.js';

/** Accordion under many-submit leaf when Load already reports N > 0. */
export function manySubmitHistoryVisible(
	submit: TaskSubmit | undefined,
	submissionCount: number | undefined
): boolean {
	return submit === 'many' && (submissionCount ?? 0) > 0;
}

export function formatSubmissionCount(count: number): string {
	if (count === 1) return m.submission_count_one({ count });
	return m.submission_count_many({ count });
}

export function taskStatusLabel(status: TaskStatus): string {
	switch (status) {
		case 'unavailable':
			return m.status_unavailable();
		case 'available':
			return m.status_available();
		case 'actionable':
			return m.status_actionable();
		case 'done':
			return m.status_done();
	}
}

export function formatFactValue(value: unknown): string {
	if (value === null || value === undefined) return '—';
	if (typeof value === 'string') return value;
	if (typeof value === 'number' || typeof value === 'boolean') return String(value);
	try {
		return JSON.stringify(value);
	} catch {
		return String(value);
	}
}
