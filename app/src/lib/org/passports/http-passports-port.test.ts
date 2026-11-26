import { describe, expect, test, vi } from 'vitest';

import { createHttpPassportsPort } from './http-passports-port.js';
import { PassportsLoadError } from './passports-port.js';

describe('createHttpPassportsPort', () => {
	test('loads passports for stream instance', async () => {
		const fetchFn = vi.fn(async () =>
			Response.json({
				passports: [
					{
						subjectId: 'CO-1',
						label: 'Carico',
						issuedAt: '2026-11-04T14:00:00Z',
						digitalLink: '/01/1/10/a/21/CO-1'
					}
				]
			})
		);
		const port = createHttpPassportsPort({ fetch: fetchFn as unknown as typeof fetch });
		const got = await port.load({ stream: 'pv-recovery', instance: '261102-1' });
		expect(fetchFn).toHaveBeenCalledWith(
			'/api/org/passports?stream=pv-recovery&instance=261102-1',
			expect.objectContaining({ method: 'GET', credentials: 'include' })
		);
		expect(got.passports).toHaveLength(1);
		expect(got.passports[0]?.subjectId).toBe('CO-1');
	});

	test('throws PassportsLoadError on non-OK', async () => {
		const fetchFn = vi.fn(async () => new Response('nope', { status: 500 }));
		const port = createHttpPassportsPort({ fetch: fetchFn as unknown as typeof fetch });
		await expect(port.load({ stream: 'pv-recovery', instance: 'x' })).rejects.toBeInstanceOf(
			PassportsLoadError
		);
	});
});
