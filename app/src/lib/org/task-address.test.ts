import { describe, expect, test } from 'vitest';

import {
	isProvisionalJobInstance,
	normalizeJobInstanceId,
	normalizeTaskAddress,
	taskAddressKey,
	taskRouteInstanceSegment,
	wireJobInstanceId
} from './task-address.js';
import { PROVISIONAL_JOB_INSTANCE_ID } from './types.js';

describe('task-address', () => {
	test('treats empty and new as provisional', () => {
		expect(isProvisionalJobInstance('')).toBe(true);
		expect(isProvisionalJobInstance(PROVISIONAL_JOB_INSTANCE_ID)).toBe(true);
		expect(isProvisionalJobInstance('NIR-1')).toBe(false);
	});

	test('normalize collapses provisional route segments to empty Domain id', () => {
		expect(normalizeJobInstanceId('')).toBe('');
		expect(normalizeJobInstanceId('new')).toBe('');
		expect(normalizeJobInstanceId('NIR-1')).toBe('NIR-1');
	});

	test('normalizeTaskAddress from route coords', () => {
		expect(
			normalizeTaskAddress({
				jobId: 'intake',
				jobInstanceId: 'new',
				taskId: 'register'
			})
		).toEqual({
			jobId: 'intake',
			jobInstanceId: '',
			taskId: 'register',
			provisionalJob: true
		});
		expect(
			normalizeTaskAddress({
				jobId: 'intake',
				jobInstanceId: 'NIR-1',
				taskId: 'register'
			})
		).toEqual({
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'register',
			provisionalJob: false
		});
	});

	test('addressKey and wire use new for provisional', () => {
		const provisional = normalizeTaskAddress({
			jobId: 'intake',
			jobInstanceId: '',
			taskId: 'register'
		});
		expect(wireJobInstanceId(provisional)).toBe('new');
		expect(taskAddressKey(provisional)).toBe('intake|new|register');

		const concrete = normalizeTaskAddress({
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'register'
		});
		expect(wireJobInstanceId(concrete)).toBe('NIR-1');
		expect(taskAddressKey(concrete)).toBe('intake|NIR-1|register');
	});

	test('empty and new route segments share the same address key', () => {
		const fromEmpty = taskAddressKey(
			normalizeTaskAddress({ jobId: 'intake', jobInstanceId: '', taskId: 'register' })
		);
		const fromNew = taskAddressKey(
			normalizeTaskAddress({ jobId: 'intake', jobInstanceId: 'new', taskId: 'register' })
		);
		expect(fromEmpty).toBe(fromNew);
	});

	test('taskRouteInstanceSegment maps Domain empty to new', () => {
		expect(taskRouteInstanceSegment('')).toBe('new');
		expect(taskRouteInstanceSegment('NIR-1')).toBe('NIR-1');
	});
});
