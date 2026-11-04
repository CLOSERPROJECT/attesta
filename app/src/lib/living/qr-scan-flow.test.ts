import { describe, expect, test, vi } from 'vitest';

import { runQrScan } from './qr-scan-flow.js';
import { QrResolveHttpError } from './qr-resolve-port.js';

describe('runQrScan', () => {
	test('seeds draft then navigates on one candidate', async () => {
		const drafts = new Map<string, Record<string, unknown>>();
		const navigate = vi.fn();
		const toast = vi.fn();
		const result = await runQrScan(
			{
				resolve: async () => ({
					navigate: {
						jobId: 'line',
						jobInstanceId: 'L1',
						groupId: 'palletWeight',
						groupInstanceId: 'new',
						taskId: 'weigh'
					},
					seed: { palletId: 'P1' }
				}),
				drafts: {
					getDraft: (path, key) => drafts.get(`${path}::${key}`) ?? null,
					setDraft: (path, key, value) => {
						drafts.set(`${path}::${key}`, value);
					}
				},
				navigate,
				toast
			},
			'https://plant.example/app?entityId=P1'
		);
		expect(result).toBe('navigated');
		expect(drafts.get('line/palletWeight/new/weigh::L1::palletWeight::')).toEqual({
			palletId: 'P1'
		});
		expect(navigate).toHaveBeenCalledWith({
			kind: 'task',
			jobId: 'line',
			jobInstanceId: 'L1',
			groupId: 'palletWeight',
			groupInstanceId: 'new',
			taskId: 'weigh'
		});
		expect(toast).not.toHaveBeenCalled();
	});

	test('toasts server error message without navigating', async () => {
		const navigate = vi.fn();
		const toast = vi.fn();
		const result = await runQrScan(
			{
				resolve: async () => ({
					error: { code: 'many', message: 'Più destinazioni possibili' }
				}),
				drafts: {
					getDraft: () => null,
					setDraft: () => undefined
				},
				navigate,
				toast
			},
			'P1'
		);
		expect(result).toBe('toasted');
		expect(toast).toHaveBeenCalledWith('Più destinazioni possibili');
		expect(navigate).not.toHaveBeenCalled();
	});

	test('toasts transport failure', async () => {
		const toast = vi.fn();
		await runQrScan(
			{
				resolve: async () => {
					throw new QrResolveHttpError('living qr-resolve failed (500)', 500);
				},
				drafts: {
					getDraft: () => null,
					setDraft: () => undefined
				},
				navigate: vi.fn(),
				toast
			},
			'P1'
		);
		expect(toast).toHaveBeenCalledWith('living qr-resolve failed (500)');
	});
});
