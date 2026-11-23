import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import {
	navPlaceKindFromRoute,
	orgHomePath,
	orgMyDashboardPath,
	orgMyInstancePath,
	orgPathSegments,
	orgPlayPath,
	orgRouteCoordsFromPath,
	orgTaskPath,
	resolveNavPlace,
	streamInstanceLabel
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
					tasks: { register: 'actionable', palletWeight: 'unavailable' }
				}
			]
		}
	]
};

describe('§6 Routes table (pathname after /app base strip)', () => {
	test('/app → app-root (redirect to /my)', () => {
		expect(navPlaceKindFromRoute({ pathname: '/' })).toBe('app-root');
		expect(navPlaceKindFromRoute({ pathname: '' })).toBe('app-root');
		expect(orgRouteCoordsFromPath({ pathname: '/' })).toEqual({
			stream: '',
			instance: '',
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		});
	});

	test('/app/{stream} → stream (cold scan or dashboard redirect)', () => {
		expect(navPlaceKindFromRoute({ pathname: '/recycling' })).toBe('stream');
		expect(navPlaceKindFromRoute({ pathname: '/recycling/' })).toBe('stream');
		expect(orgRouteCoordsFromPath({ pathname: '/recycling' })).toEqual({
			stream: 'recycling',
			instance: '',
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		});
		expect(orgMyDashboardPath('recycling')).toBe('/my/streams/org-recycling/');
	});

	test('/app/{stream}/new → play (anchor start Task)', () => {
		expect(navPlaceKindFromRoute({ pathname: '/recycling/new' })).toBe('play');
		expect(orgRouteCoordsFromPath({ pathname: '/recycling/new' })).toEqual({
			stream: 'recycling',
			instance: '',
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		});
		expect(orgPlayPath('recycling')).toBe('/recycling/new');
	});

	test('/app/{stream}/i/{instance} → home (instance Worklist)', () => {
		expect(navPlaceKindFromRoute({ pathname: '/recycling/i/T1' })).toBe('home');
		expect(orgRouteCoordsFromPath({ pathname: '/recycling/i/T1' })).toEqual({
			stream: 'recycling',
			instance: 'T1',
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		});
		expect(orgHomePath('recycling', 'T1')).toBe('/recycling/i/T1');
	});

	test('/app/{stream}/i/{instance}/{job}/{jobInstance}/{task} → task', () => {
		expect(
			navPlaceKindFromRoute({
				pathname: '/recycling/i/T1/intake/NIR-2/register'
			})
		).toBe('task');
		expect(
			orgRouteCoordsFromPath({
				pathname: '/recycling/i/T1/intake/NIR-2/register'
			})
		).toEqual({
			stream: 'recycling',
			instance: 'T1',
			jobId: 'intake',
			jobInstanceId: 'NIR-2',
			taskId: 'register'
		});
		expect(orgTaskPath('recycling', 'T1', 'intake', 'NIR-2', 'register')).toBe(
			'/recycling/i/T1/intake/NIR-2/register'
		);
	});

	test('provisional Task under instance uses new sentinel', () => {
		expect(
			orgRouteCoordsFromPath({
				pathname: '/recycling/i/T1/lot_processing/new/startLot'
			})
		).toEqual({
			stream: 'recycling',
			instance: 'T1',
			jobId: 'lot_processing',
			jobInstanceId: 'new',
			taskId: 'startLot'
		});
		expect(
			navPlaceKindFromRoute({
				pathname: '/recycling/i/T1/lot_processing/new/startLot'
			})
		).toBe('task');
	});

	test('no archive or overview place kinds', () => {
		// Legacy /archive and /{job}/{instance} shapes are not place kinds anymore.
		expect(navPlaceKindFromRoute({ pathname: '/archive' })).not.toBe('archive');
		expect(navPlaceKindFromRoute({ pathname: '/intake/NIR-2' })).toBe('app-root');
		const kinds = [
			navPlaceKindFromRoute({ pathname: '/' }),
			navPlaceKindFromRoute({ pathname: '/recycling' }),
			navPlaceKindFromRoute({ pathname: '/recycling/new' }),
			navPlaceKindFromRoute({ pathname: '/recycling/i/T1' }),
			navPlaceKindFromRoute({ pathname: '/recycling/i/T1/intake/a/b' }),
			navPlaceKindFromRoute({ pathname: '/archive' }),
			navPlaceKindFromRoute({ pathname: '/intake/NIR-2' })
		];
		expect(kinds).not.toContain('archive');
		expect(kinds).not.toContain('overview');
	});

	test('trailing slash on root still → app-root', () => {
		expect(navPlaceKindFromRoute({ pathname: '//' })).toBe('app-root');
	});

	test('orgPathSegments strips trailing slash', () => {
		expect(orgPathSegments('/recycling/i/T1/')).toEqual(['recycling', 'i', 'T1']);
	});

	test('orgMyInstancePath is SPA instance home', () => {
		expect(orgMyInstancePath('recycling', 'T1')).toBe('/recycling/i/T1');
	});

	test('streamInstanceLabel reads anchor instance label from Worklist', () => {
		expect(streamInstanceLabel(worklist, 'NIR-2')).toBe('Camion B');
		expect(streamInstanceLabel(worklist, 'missing')).toBe('missing');
	});
});

describe('resolveNavPlace', () => {
	test('home: Home label, no back, instance home target', () => {
		expect(
			resolveNavPlace({
				kind: 'home',
				stream: 'recycling',
				instance: 'T1',
				jobId: '',
				jobInstanceId: '',
				taskId: '',
				provisionalJob: false,
				worklist
			})
		).toEqual({
			showBack: false,
			backTarget: { kind: 'home', stream: 'recycling', instance: 'T1' },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		});
	});

	test('play: Home label, no back (board owns back to my stream)', () => {
		expect(
			resolveNavPlace({
				kind: 'play',
				stream: 'recycling',
				instance: '',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'register',
				provisionalJob: true,
				worklist
			})
		).toEqual({
			showBack: false,
			backTarget: { kind: 'myStream', stream: 'recycling' },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		});
	});

	test('task provisional Job: task title only, back instance home', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				stream: 'recycling',
				instance: 'T1',
				jobId: 'intake',
				jobInstanceId: '',
				taskId: 'register',
				provisionalJob: true,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home', stream: 'recycling', instance: 'T1' },
			primaryLabel: 'Registrazione',
			jobTitleLine: 'Camion'
		});
	});

	test('task minted: job · instance muted, task primary, back instance home', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				stream: 'recycling',
				instance: 'NIR-2',
				jobId: 'intake',
				jobInstanceId: 'NIR-2',
				taskId: 'register',
				provisionalJob: false,
				worklist
			})
		).toEqual({
			showBack: true,
			backTarget: { kind: 'home', stream: 'recycling', instance: 'NIR-2' },
			primaryLabel: 'Registrazione',
			jobTitleLine: 'Camion · Camion B'
		});
	});

	test('task falls back to taskId when chrome title absent', () => {
		expect(
			resolveNavPlace({
				kind: 'task',
				stream: 'recycling',
				instance: 'T1',
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
					instances: [
						{
							id: 'NIR-1',
							label: 'NIR-1',
							tasks: { codiceFir: 'actionable' }
						}
					]
				}
			]
		};
		const place = resolveNavPlace({
			kind: 'task',
			stream: 'recycling',
			instance: 'NIR-1',
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
