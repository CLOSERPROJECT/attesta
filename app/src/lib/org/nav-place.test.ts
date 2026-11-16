import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import {
	orgRouteCoordsFromPath,
	navPlaceKindFromRoute,
	resolveNavPlace
} from './nav-place.js';
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
					tasks: { register: 'actionable', palletWeight: 'locked' }
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
					tasks: { only: 'actionable' }
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

	test('pathname /archive → archive (not a Job id)', () => {
		expect(navPlaceKindFromRoute({ pathname: '/archive' })).toBe('archive');
		expect(navPlaceKindFromRoute({ pathname: '/archive/' })).toBe('archive');
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

	test('home: Home label, no back, no job title line', () => {
		expect(
			resolveNavPlace({
				kind: 'home',
				jobId: '',
				jobInstanceId: '',
				taskId: '',
				provisionalJob: false,
				worklist
			})
		).toEqual({
			showBack: false,
			backTarget: { kind: 'home' },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		});
	});

	test('archive: Archive label, back home, no job title line', () => {
		expect(
			resolveNavPlace({
				kind: 'archive',
				jobId: '',
				jobInstanceId: '',
				taskId: '',
				provisionalJob: false,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: m.archive(),
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
				provisionalJob: false,
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
				provisionalJob: false,
				worklist
			}).jobTitleLine
		).toBeNull();
	});

	test('task provisional Job: task title only, back home', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'register',
				provisionalJob: true,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: 'Registrazione',
			jobTitleLine: 'Camion'
		});
	});

	test('task minted: job · instance muted, task primary, back home', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: 'NIR-2',
				taskId: 'register',
				provisionalJob: false,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: 'Registrazione',
			jobTitleLine: 'Camion · Camion B'
		});
	});

	test('task falls back to taskId when chrome title absent', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'unknownTask',
				provisionalJob: true,
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
		const place = resolveNavPlace({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'codiceFir',
			provisionalJob: false,
			worklist: wl
		});
		expect(place.primaryLabel).toBe('codiceFir');
		expect(place.jobTitleLine).toBe('NIR-1');
	});
});

describe('orgRouteCoordsFromPath', () => {
	test('flat Can-start 3-seg Task coords', () => {
		expect(orgRouteCoordsFromPath({ pathname: '/intake/new/truckArrive' })).toEqual({
			jobId: 'intake',
			jobInstanceId: 'new',
			taskId: 'truckArrive'
		});
	});

	test('flat minted 3-seg Task coords', () => {
		expect(
			orgRouteCoordsFromPath({ pathname: '/intake/261102-1st-2/weightCheck' })
		).toEqual({
			jobId: 'intake',
			jobInstanceId: '261102-1st-2',
			taskId: 'weightCheck'
		});
	});

	test('legacy 5-seg nested group path is rejected', () => {
		expect(
			orgRouteCoordsFromPath({ pathname: '/intake/NIR-2/lots/new/register' })
		).toEqual({
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		});
	});

	test('2-seg overview clears task coords', () => {
		expect(orgRouteCoordsFromPath({ pathname: '/intake/NIR-2' })).toEqual({
			jobId: 'intake',
			jobInstanceId: 'NIR-2',
			taskId: ''
		});
	});
});