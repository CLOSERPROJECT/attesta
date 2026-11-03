import { describe, expect, test } from 'vitest';

import { navPlaceKindFromRoute, resolveNavPlace } from './nav-place.js';
import type { Worklist } from './types.js';

const worklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: true,
			repeatable: true,
			title: 'Camion',
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [
				{ id: 'register', title: 'Registrazione', roles: ['administration'] },
				{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
			],
			instances: [
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' },
					viewerAvailable: ['register']
				}
			]
		},
		{
			jobId: 'untitled',
			canStart: true,
			repeatable: false,
			taskIds: ['only'],
			tasks: [{ id: 'only', title: 'Solo', roles: ['administration'] }],
			instances: [
				{
					id: 'u1',
					label: 'U-1',
					tasks: { only: 'available' },
					viewerAvailable: ['only']
				}
			]
		}
	]
};

describe('navPlaceKindFromRoute', () => {
	test('pathname / → home', () => {
		expect(navPlaceKindFromRoute({ pathname: '/' })).toBe('home');
	});

	test('trailing slash on root still → home', () => {
		// stripBase('/app/') yields '/'; a leftover '//' must not become overview
		expect(navPlaceKindFromRoute({ pathname: '//' })).toBe('home');
	});

	test('empty pathname → home', () => {
		expect(navPlaceKindFromRoute({ pathname: '' })).toBe('home');
	});

	test('3 path segments → task', () => {
		expect(navPlaceKindFromRoute({ pathname: '/intake/NIR-2/register' })).toBe('task');
	});

	test('2 path segments → overview', () => {
		expect(navPlaceKindFromRoute({ pathname: '/intake/NIR-2' })).toBe('overview');
	});

	test('2-segment overview ignores stale taskId in URL params (pathname only)', () => {
		// Regression: sv-router may keep taskId after leaving a task route.
		expect(navPlaceKindFromRoute({ pathname: '/intake/NIR-261102-NO-1' })).toBe(
			'overview'
		);
	});

	test('trailing slash on overview still → overview', () => {
		expect(navPlaceKindFromRoute({ pathname: '/intake/NIR-2/' })).toBe('overview');
	});
});

describe('resolveNavPlace', () => {
	test('home: Home label, no back, no job title line', () => {
		expect(
			resolveNavPlace({
				kind: 'home',
				jobId: '',
				jobInstanceId: '',
				taskId: '',
				provisional: false,
				worklist
			})
		).toEqual({
			showBack: false,
			backTarget: { kind: 'home' },
			primaryLabel: 'Home',
			jobTitleLine: null
		});
	});

	test('overview: instance label only (no ·), back home, authored job title', () => {
		expect(
			resolveNavPlace({
				kind: 'overview',
				jobId: 'intake',
				jobInstanceId: 'NIR-2',
				taskId: 'register',
				provisional: false,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: 'Camion B',
			jobTitleLine: 'Camion'
		});
	});

	test('overview: omits job title line when Job.title missing (never raw id)', () => {
		expect(
			resolveNavPlace({
				kind: 'overview',
				jobId: 'untitled',
				jobInstanceId: 'u1',
				taskId: '',
				provisional: false,
				worklist
			}).jobTitleLine
		).toBeNull();
	});

	test('task provisional: task title only, back home', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'register',
				provisional: true,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: 'Registrazione',
			jobTitleLine: 'Camion'
		});
	});

	test('task minted: instance · task, back overview', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: 'NIR-2',
				taskId: 'register',
				provisional: false,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-2' },
			primaryLabel: 'Camion B · Registrazione',
			jobTitleLine: 'Camion'
		});
	});

	test('task falls back to taskId when chrome title absent', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'unknownTask',
				provisional: true,
				worklist
			}).primaryLabel
		).toBe('unknownTask');
	});

	test('task never paints literal undefined when title field missing on chrome', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['codiceFir'],
					tasks: [
						{
							id: 'codiceFir',
							title: undefined as unknown as string,
							roles: ['administration']
						}
					],
					instances: []
				}
			]
		};
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: 'NIR-1',
				taskId: 'codiceFir',
				provisional: false,
				worklist: wl
			}).primaryLabel
		).toBe('NIR-1 · codiceFir');
	});
});
