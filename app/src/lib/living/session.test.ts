import { describe, expect, test, vi } from 'vitest';

import { LivingSession } from './session.svelte.js';
import type { LoadDomainPort } from './load-port.js';
import { emptyDomain, type LoadResult } from './types.js';

const sample: LoadResult = {
	org: 'feragame',
	activeConfigVersion: '1',
	domain: emptyDomain(),
	config: {
		id: 'truck-loop',
		org: 'feragame',
		version: '1',
		jobs: {
			intake: {
				repeatable: true,
				label: '$load.id',
				startLabel: 'Nuovo camion',
				tasks: { register: {} }
			},
			line: {
				repeatable: true,
				label: 'line',
				startLabel: 'Nuovo turno',
				tasks: { shift: {} }
			}
		}
	}
};

describe('LivingSession', () => {
	test('hydrate loads Domain + Active Config via LoadDomainPort', async () => {
		const load = vi.fn(async () => sample) satisfies LoadDomainPort['load'];
		const session = new LivingSession({
			load: { load },
			hydrate: false
		});
		expect(session.result).toBeNull();
		await session.hydrate();
		expect(load).toHaveBeenCalledOnce();
		expect(session.error).toBeNull();
		expect(session.result?.org).toBe('feragame');
		expect(session.jobIds).toEqual(['intake', 'line']);
		expect(session.job('intake')?.startLabel).toBe('Nuovo camion');
	});

	test('hydrate surfaces port errors without leaving stale result', async () => {
		const session = new LivingSession({
			load: {
				load: async () => {
					throw new Error('boom');
				}
			},
			hydrate: false
		});
		session.result = sample;
		await session.hydrate();
		expect(session.result).toBeNull();
		expect(session.error).toBe('boom');
	});
});
