import { describe, expect, test } from 'vitest';

import {
	factTitle,
	formatSubmissionTimestamp,
	formatSubmissionUser,
	unionFactKeys
} from './submission-table.js';
import type { TaskRecap } from './task-recap-port.js';

function recap(
	facts: Record<string, unknown>,
	titles?: Record<string, string>
): TaskRecap {
	return {
		jobId: 'j',
		jobInstanceId: 'i',
		taskId: 't',
		timestamp: '2024-01-01T00:00:00Z',
		userId: 'u',
		facts,
		titles,
		links: []
	};
}

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

	test('unionFactKeys walks items in order and appends unseen keys', () => {
		expect(
			unionFactKeys([
				recap({ b: 1, a: 2 }),
				recap({ a: 3, c: 4 }),
				recap({ d: 5, b: 6 })
			])
		).toEqual(['b', 'a', 'c', 'd']);
	});

	test('unionFactKeys skips empty keys', () => {
		expect(unionFactKeys([recap({ '': 1, x: 2 })])).toEqual(['x']);
	});

	test('factTitle prefers first item with a title for the key', () => {
		const items = [
			recap({ a: 1 }),
			recap({ a: 2 }, { a: 'Alpha' }),
			recap({ a: 3 }, { a: 'Other' })
		];
		expect(factTitle(items, 'a')).toBe('Alpha');
		expect(factTitle(items, 'missing')).toBe('missing');
	});
});
