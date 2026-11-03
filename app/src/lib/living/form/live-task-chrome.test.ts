import { describe, expect, test } from 'vitest';

import {
	evaluateChromeEntry,
	formatChromeEvalValue
} from './live-task-chrome.svelte.js';
import { operatorFromLoad } from './operator-identity.js';

describe('formatChromeEvalValue', () => {
	test('undefined → chrome-failed', () => {
		expect(formatChromeEvalValue(undefined)).toEqual({
			value: null,
			error: 'chrome-failed'
		});
	});

	test('string value paints', () => {
		expect(formatChromeEvalValue('Ada')).toEqual({ value: 'Ada', error: null });
	});
});

describe('evaluateChromeEntry $operator', () => {
	test('with getOperator displayName resolves $operator.displayName', async () => {
		const operator = operatorFromLoad({
			org: 'feragame',
			userId: 'u1',
			displayName: 'Ada',
			roles: ['administration']
		});
		const got = await evaluateChromeEntry(
			'$operator.displayName',
			{},
			{},
			undefined,
			operator
		);
		expect(got).toEqual({ value: 'Ada', error: null });
	});

	test('without operator → chrome-failed', async () => {
		const got = await evaluateChromeEntry('$operator.displayName', {}, {}, undefined);
		expect(got).toEqual({ value: null, error: 'chrome-failed' });
	});
});

describe('operatorFromLoad', () => {
	test('maps Load viewer fields', () => {
		expect(
			operatorFromLoad({
				org: 'feragame',
				userId: 'u1',
				displayName: 'Ada',
				roles: ['administration']
			})
		).toEqual({
			userId: 'u1',
			displayName: 'Ada',
			org: 'feragame',
			roles: ['administration']
		});
	});

	test('null Load → undefined', () => {
		expect(operatorFromLoad(null)).toBeUndefined();
	});
});
