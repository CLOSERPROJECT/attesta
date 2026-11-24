import { describe, expect, test, vi } from 'vitest';

import { createDomainChangeTracker } from './domain-change-tracker.js';

describe('domain change tracker', () => {
	test('ignores remote revisions before the first reset', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.remote(3);

		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBeNull();
	});

	test('reset establishes the baseline and remote emits only newer revisions', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.remote(5);
		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBe(5);

		tracker.remote(6);
		expect(onForeign).toHaveBeenCalledTimes(1);
		expect(onForeign).toHaveBeenLastCalledWith(6);
		expect(tracker.known).toBe(6);
	});

	test('remote gaps are just newer', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.remote(7);

		expect(onForeign).toHaveBeenCalledTimes(1);
		expect(onForeign).toHaveBeenLastCalledWith(7);
		expect(tracker.known).toBe(7);
	});

	test('own echo before the Commit response is not foreign', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.remote(6);
		tracker.commitSettled(6);

		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBe(6);
	});

	test('own echo after the Commit response is not foreign', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.commitSettled(6);
		tracker.remote(6);

		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBe(6);
	});

	test('held foreign revision emits once when the Commit settles', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.remote(6);
		tracker.remote(7);
		tracker.commitSettled(7);

		expect(onForeign).toHaveBeenCalledTimes(1);
		expect(onForeign).toHaveBeenLastCalledWith(7);
		expect(tracker.known).toBe(7);
	});

	test('settled revision past the baseline reports a write never heard about', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.commitSettled(7);

		expect(onForeign).toHaveBeenCalledTimes(1);
		expect(onForeign).toHaveBeenLastCalledWith(7);
		expect(tracker.known).toBe(7);

		tracker.remote(6);
		expect(onForeign).toHaveBeenCalledTimes(1);
	});

	test('failed Commit with a held remote revision emits it', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.remote(6);
		tracker.commitSettled(null);

		expect(onForeign).toHaveBeenCalledTimes(1);
		expect(onForeign).toHaveBeenLastCalledWith(6);
		expect(tracker.known).toBe(6);
	});

	test('failed Commit with no held revision emits nothing', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.commitSettled(null);

		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBe(5);
	});

	test('scope reload mid-Commit resets the baseline without emitting', () => {
		const onForeign = vi.fn();
		const tracker = createDomainChangeTracker(onForeign);

		tracker.reset(5);
		tracker.commitStarted();
		tracker.reset(9);
		tracker.commitSettled(10);

		expect(onForeign).not.toHaveBeenCalled();
		expect(tracker.known).toBe(10);
	});
});
