import { describe, expect, test } from 'vitest';

import { normalizeTaskAddress } from '../task-address.js';
import {
	addressFromBoardTask,
	boardHrefWithTask,
	boardTaskFromAddress,
	encodeBoardTaskParam,
	focusMatchesAddress,
	parseBoardTaskParam,
	parseTaskHref,
	readBoardTaskFromSearch
} from './board-url.js';

describe('board-url ?task= sync', () => {
	test('encode/parse round-trip; provisional uses new', () => {
		expect(
			encodeBoardTaskParam({
				jobId: 'intake',
				jobInstanceId: 'NIR-1',
				taskId: 'register'
			})
		).toBe('intake/NIR-1/register');
		expect(
			encodeBoardTaskParam({
				jobId: 'lot_processing',
				jobInstanceId: '',
				taskId: 'startLot'
			})
		).toBe('lot_processing/new/startLot');
		expect(parseBoardTaskParam('intake/NIR-1/register')).toEqual({
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'register'
		});
		expect(parseBoardTaskParam('bad')).toBeNull();
	});

	test('readBoardTaskFromSearch accepts string and params', () => {
		expect(readBoardTaskFromSearch('?task=intake/NIR-1/register')).toEqual({
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'register'
		});
		expect(
			readBoardTaskFromSearch(new URLSearchParams('task=a/b/c'))
		).toEqual({ jobId: 'a', jobInstanceId: 'b', taskId: 'c' });
		expect(readBoardTaskFromSearch({ task: 'x/y/z' })).toEqual({
			jobId: 'x',
			jobInstanceId: 'y',
			taskId: 'z'
		});
	});

	test('boardHrefWithTask sets and clears param', () => {
		expect(
			boardHrefWithTask('/recycling/i/NIR-1', {
				jobId: 'intake',
				jobInstanceId: 'NIR-1',
				taskId: 'register'
			})
		).toBe('/recycling/i/NIR-1?task=intake%2FNIR-1%2Fregister');
		expect(
			boardHrefWithTask('/recycling/i/NIR-1?task=intake/NIR-1/register', null)
		).toBe('/recycling/i/NIR-1');
	});

	test('address helpers + focus match', () => {
		const addr = normalizeTaskAddress({
			stream: 'recycling',
			instance: 'NIR-1',
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			taskId: 'register'
		});
		const focus = boardTaskFromAddress(addr);
		expect(addressFromBoardTask('recycling', 'NIR-1', focus)).toEqual(addr);
		expect(focusMatchesAddress(focus, addr)).toBe(true);
		expect(
			focusMatchesAddress(
				{ jobId: 'intake', jobInstanceId: 'other', taskId: 'register' },
				addr
			)
		).toBe(false);
	});

	test('parseTaskHref strips /app and seed.*', () => {
		const parsed = parseTaskHref(
			'/app/recycling/i/NIR-1/lot_processing/new/startLot?seed.lotId=L1',
			'recycling',
			'NIR-1'
		);
		expect(parsed?.address).toEqual(
			normalizeTaskAddress({
				stream: 'recycling',
				instance: 'NIR-1',
				jobId: 'lot_processing',
				jobInstanceId: 'new',
				taskId: 'startLot'
			})
		);
		expect(parsed?.seed).toEqual({ lotId: 'L1' });
	});

	test('parseTaskHref accepts Play boot …/new/{job}/{ji}/{task}', () => {
		const parsed = parseTaskHref(
			'/app/recycling/new/intake/new/truckArrive',
			'recycling',
			''
		);
		expect(parsed?.address).toEqual(
			normalizeTaskAddress({
				stream: 'recycling',
				instance: '',
				jobId: 'intake',
				jobInstanceId: 'new',
				taskId: 'truckArrive'
			})
		);
		expect(parsed?.seed).toEqual({});
	});

	test('parseTaskHref Play shape strips seed.*', () => {
		const parsed = parseTaskHref(
			'/app/recycling/new/lot_processing/new/startLot?seed.lotId=L1',
			'recycling'
		);
		expect(parsed?.address).toEqual(
			normalizeTaskAddress({
				stream: 'recycling',
				instance: '',
				jobId: 'lot_processing',
				jobInstanceId: 'new',
				taskId: 'startLot'
			})
		);
		expect(parsed?.seed).toEqual({ lotId: 'L1' });
	});
});
