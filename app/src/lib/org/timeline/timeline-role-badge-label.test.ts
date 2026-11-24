import { describe, expect, test } from 'vitest';

import {
	timelineRoleBadgeLabel,
	timelineRoleBadgesVisible
} from './timeline-role-badge-label.js';

describe('timelineRoleBadgeLabel', () => {
	test('prefers trimmed label over id', () => {
		expect(timelineRoleBadgeLabel({ id: 'dep1', label: '  Depot  ', palette: 'blue' })).toBe(
			'Depot'
		);
	});

	test('falls back to id when label empty', () => {
		expect(timelineRoleBadgeLabel({ id: 'dep1', label: '  ', palette: 'fallback' })).toBe(
			'dep1'
		);
		expect(timelineRoleBadgeLabel({ id: 'dep1', label: '', palette: 'cyan' })).toBe('dep1');
	});
});

describe('timelineRoleBadgesVisible', () => {
	const badges = [
		{ id: 'a', label: 'A', palette: 'blue' },
		{ id: 'b', label: 'B', palette: 'emerald' }
	];

	test('empty when showAll is off', () => {
		expect(timelineRoleBadgesVisible(false, badges)).toEqual([]);
	});

	test('returns API order when showAll is on', () => {
		expect(timelineRoleBadgesVisible(true, badges)).toEqual(badges);
	});

	test('empty when badges missing or empty', () => {
		expect(timelineRoleBadgesVisible(true, undefined)).toEqual([]);
		expect(timelineRoleBadgesVisible(true, [])).toEqual([]);
	});
});
