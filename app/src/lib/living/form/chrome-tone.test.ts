import { describe, expect, test } from 'vitest';
import { isChromeTone, resolveChromeTone } from './chrome-tone.js';

describe('resolveChromeTone', () => {
	test('static tone when bands absent', () => {
		expect(resolveChromeTone(12, { tone: 'info' })).toBe('info');
		expect(resolveChromeTone('x', { tone: 'warning' })).toBe('warning');
		expect(resolveChromeTone(1, {})).toBeNull();
	});

	test('first band where value ≥ min wins (author high→low)', () => {
		const bands = [
			{ min: 98, tone: 'green' as const },
			{ min: 90, tone: 'warning' as const },
			{ min: 0, tone: 'danger' as const }
		];
		expect(resolveChromeTone(99, { bands })).toBe('green');
		expect(resolveChromeTone(98, { bands })).toBe('green');
		expect(resolveChromeTone(95, { bands })).toBe('warning');
		expect(resolveChromeTone(90, { bands })).toBe('warning');
		expect(resolveChromeTone(50, { bands })).toBe('danger');
		expect(resolveChromeTone(0, { bands })).toBe('danger');
	});

	test('non-numeric → no band tone (even with static tone present)', () => {
		const bands = [{ min: 0, tone: 'danger' as const }];
		expect(resolveChromeTone('98', { tone: 'info', bands })).toBeNull();
		expect(resolveChromeTone(undefined, { tone: 'info', bands })).toBeNull();
		expect(resolveChromeTone(Number.NaN, { bands })).toBeNull();
	});

	test('empty bands fall through to static tone', () => {
		expect(resolveChromeTone(1, { tone: 'green', bands: [] })).toBe('green');
	});
});

describe('isChromeTone', () => {
	test('accepts catalog tones only', () => {
		expect(isChromeTone('info')).toBe(true);
		expect(isChromeTone('green')).toBe(true);
		expect(isChromeTone('red')).toBe(false);
		expect(isChromeTone(1)).toBe(false);
	});
});
