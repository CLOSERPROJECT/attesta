import { describe, expect, test, vi } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import { runQrScan } from './qr-scan-flow.js';
import { QrResolveHttpError } from './qr-resolve-port.js';

describe('runQrScan', () => {
	test('seeds draft then navigates on one candidate (flat weigh)', async () => {
		const drafts = new Map<string, Record<string, unknown>>();
		const navigate = vi.fn();
		const toast = vi.fn();
		const resolve = vi.fn(async () => ({
			navigate: {
				jobId: 'line',
				jobInstanceId: 'L1',
				taskId: 'weigh',
				streamInstanceId: '261102-1'
			},
			seed: { palletId: 'P1' }
		}));
		const result = await runQrScan(
			{
				resolve,
				stream: 'recycling',
				instance: '261102-1',
				drafts: {
					getDraft: (path, key) => drafts.get(`${path}::${key}`) ?? null,
					setDraft: (path, key, value) => {
						drafts.set(`${path}::${key}`, value);
					}
				},
				navigate,
				toast
			},
			'https://plant.example/app/recycling?entityId=P1'
		);
		expect(result).toBe('navigated');
		expect(resolve).toHaveBeenCalledWith(
			'https://plant.example/app/recycling?entityId=P1',
			{ stream: 'recycling', instance: '261102-1' }
		);
		expect(drafts.get('line/weigh::L1')).toEqual({
			palletId: 'P1'
		});
		expect(navigate).toHaveBeenCalledWith({
			kind: 'task',
			stream: 'recycling',
			instance: '261102-1',
			jobId: 'line',
			jobInstanceId: 'L1',
			taskId: 'weigh'
		});
		expect(toast).not.toHaveBeenCalled();
	});

	test('toasts switched-instance then navigates when resolve targets another instance', async () => {
		const drafts = new Map<string, Record<string, unknown>>();
		const navigate = vi.fn();
		const toast = vi.fn();
		const result = await runQrScan(
			{
				resolve: async () => ({
					navigate: {
						jobId: 'line',
						jobInstanceId: 'L2',
						taskId: 'weigh',
						streamInstanceId: '261102-2'
					},
					seed: { palletId: 'P2' }
				}),
				stream: 'recycling',
				instance: '261102-1',
				drafts: {
					getDraft: (path, key) => drafts.get(`${path}::${key}`) ?? null,
					setDraft: (path, key, value) => {
						drafts.set(`${path}::${key}`, value);
					}
				},
				navigate,
				toast
			},
			'P2'
		);
		expect(result).toBe('navigated');
		expect(toast).toHaveBeenCalledWith(m.scan_switched_instance());
		expect(navigate).toHaveBeenCalledWith({
			kind: 'task',
			stream: 'recycling',
			instance: '261102-2',
			jobId: 'line',
			jobInstanceId: 'L2',
			taskId: 'weigh'
		});
	});

	test('same-instance resolve does not toast switched-instance', async () => {
		const toast = vi.fn();
		await runQrScan(
			{
				resolve: async () => ({
					navigate: {
						jobId: 'line',
						jobInstanceId: 'L1',
						taskId: 'weigh',
						streamInstanceId: '261102-1'
					},
					seed: {}
				}),
				stream: 'recycling',
				instance: '261102-1',
				drafts: {
					getDraft: () => null,
					setDraft: () => undefined
				},
				navigate: vi.fn(),
				toast
			},
			'P1'
		);
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
				stream: 'recycling',
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
					throw new QrResolveHttpError('org qr-resolve failed (500)', 500);
				},
				stream: 'recycling',
				drafts: {
					getDraft: () => null,
					setDraft: () => undefined
				},
				navigate: vi.fn(),
				toast
			},
			'x'
		);
		expect(toast).toHaveBeenCalledWith('org qr-resolve failed (500)');
	});
});
