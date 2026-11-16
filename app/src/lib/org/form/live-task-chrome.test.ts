import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import {
	evaluateChromeEntry,
	formatChromeEvalValue
} from './live-task-chrome.svelte.js';
import { operatorFromLoad } from './operator-identity.js';
import { utcCalendarDateISO } from './schema-defaults.js';

describe('formatChromeEvalValue', () => {
	test('undefined → chrome-failed', () => {
		expect(formatChromeEvalValue(undefined)).toEqual({
			value: null,
			error: m.chrome_failed()
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
		expect(got).toEqual({ value: null, error: m.chrome_failed() });
	});
});

describe('evaluateChromeEntry $today', () => {
	test('binds UTC YYYY-MM-DD matching Commit/transform semantics', async () => {
		const today = utcCalendarDateISO();
		const got = await evaluateChromeEntry('$today', {}, {}, undefined);
		expect(got).toEqual({ value: today, error: null });
	});

	test('progressive-style dateStamp uses $today', async () => {
		const today = utcCalendarDateISO();
		const dateStamp = today.replaceAll('-', '').slice(2);
		const got = await evaluateChromeEntry(
			"$substring($replace($today, '-', ''), 2)",
			{},
			{},
			undefined
		);
		expect(got).toEqual({ value: dateStamp, error: null });
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
