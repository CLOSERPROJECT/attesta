import { describe, expect, test, vi } from 'vitest';

import { createHttpQrResolvePort } from './http-qr-resolve-port.js';
import { QrResolveHttpError } from './qr-resolve-port.js';

describe('HttpQrResolvePort', () => {
	test('POST /api/living/qr-resolve with { raw } and parses navigate+seed', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/living/qr-resolve');
			expect(init?.method).toBe('POST');
			expect(init?.credentials).toBe('include');
			expect(JSON.parse(String(init?.body))).toEqual({
				raw: 'https://plant.example/app?entityId=P1'
			});
			return new Response(
				JSON.stringify({
					navigate: {
						jobId: 'line',
						jobInstanceId: 'L1',
						groupId: 'palletWeight',
						groupInstanceId: 'new',
						taskId: 'weigh'
					},
					seed: { palletId: 'P1' }
				}),
				{ status: 200, headers: { 'Content-Type': 'application/json' } }
			);
		});

		const port = createHttpQrResolvePort({ fetch: fetchFn });
		const got = await port.resolve('https://plant.example/app?entityId=P1');
		expect(got.error).toBeUndefined();
		expect(got.navigate?.taskId).toBe('weigh');
		expect(got.seed?.palletId).toBe('P1');
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('422 toastable error body is returned, not thrown', async () => {
		const fetchFn = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: 'ineligible',
							message: 'Bancale non pesabile (assente o già pesato)'
						}
					}),
					{ status: 422, headers: { 'Content-Type': 'application/json' } }
				)
		);
		const port = createHttpQrResolvePort({ fetch: fetchFn });
		const got = await port.resolve('missing');
		expect(got.navigate).toBeUndefined();
		expect(got.error?.code).toBe('ineligible');
		expect(got.error?.message).toContain('non pesabile');
	});

	test('non-OK non-422 throws QrResolveHttpError', async () => {
		const fetchFn = vi.fn(async () => new Response('boom', { status: 500 }));
		const port = createHttpQrResolvePort({ fetch: fetchFn });
		await expect(port.resolve('x')).rejects.toBeInstanceOf(QrResolveHttpError);
	});
});
