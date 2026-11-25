import { describe, expect, test } from 'vitest';

import {
	formatSubmissionTimestamp,
	formatSubmissionUser,
	recapRowMatchesSubmissionId
} from './submission-table.js';

describe('submission-table helpers', () => {
	test('formatSubmissionTimestamp localizes a parseable ISO string', () => {
		const raw = '2024-01-15T14:30:00Z';
		const out = formatSubmissionTimestamp(raw, 'en-US');
		expect(out).not.toBe(raw);
		expect(out).not.toBe('—');
		expect(out).toMatch(/2024/);
	});

	test('formatSubmissionTimestamp empty and invalid', () => {
		expect(formatSubmissionTimestamp('')).toBe('—');
		expect(formatSubmissionTimestamp('not-a-date')).toBe('not-a-date');
	});

	test('formatSubmissionUser prefers userName over userId', () => {
		expect(formatSubmissionUser({ userId: 'id-1', userName: 'Ada' })).toBe('Ada');
		expect(formatSubmissionUser({ userId: 'id-1' })).toBe('id-1');
		expect(formatSubmissionUser({ userId: '', userName: '' })).toBe('—');
	});

	test('recapRowMatchesSubmissionId matches submissionId', () => {
		const id = '2026-11-13T12:00:01Z';
		expect(recapRowMatchesSubmissionId({ submissionId: id }, id)).toBe(true);
		expect(
			recapRowMatchesSubmissionId({ submissionId: '2026-11-13T12:00:00Z' }, id)
		).toBe(false);
		expect(recapRowMatchesSubmissionId({ submissionId: id }, undefined)).toBe(false);
		expect(recapRowMatchesSubmissionId({ submissionId: id }, '')).toBe(false);
	});
});
