import { describe, expect, test } from 'vitest';

import { normalizeTaskAddress, type TaskAddress } from '../task-address.js';
import {
	planBoardRefresh,
	type BoardRefreshFacts,
	type DomainChange
} from './board-refresh-plan.js';

function taskAddr(taskId = 'register'): TaskAddress {
	return normalizeTaskAddress({
		stream: 'recycling',
		instance: '261102-1',
		jobId: 'intake',
		jobInstanceId: '261102-1',
		taskId
	});
}

function change(
	follow: DomainChange['follow'],
	addr: TaskAddress,
	submitMany = false
): DomainChange {
	return { origin: 'own', revision: 7, addr, follow, submitMany };
}

const viewable: BoardRefreshFacts = { scopeKind: 'instance', hasView: true };
const noViewInstance: BoardRefreshFacts = { scopeKind: 'instance', hasView: false };
const noViewBoot: BoardRefreshFacts = { scopeKind: 'boot', hasView: false };

describe('planBoardRefresh', () => {
	test('stay reloads the timeline, remounts forms on submitMany, and refreshes the recap', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('stay', addr, true)], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: true,
			recapRefresh: [addr]
		});
	});

	test('stay without submitMany keeps form bundles', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('stay', addr)], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('advance with a view reloads the timeline on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('advance with a view reloads the timeline on boot scope', () => {
		const addr = taskAddr();
		expect(
			planBoardRefresh([change('advance', addr)], {
				scopeKind: 'boot',
				hasView: true
			})
		).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('advance without a view hydrates the Worklist on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], noViewInstance)).toEqual({
			timeline: false,
			worklistOnly: true,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('advance without a view does nothing extra on boot scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], noViewBoot)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('leaving only refreshes the recap on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('leaving', addr)], viewable)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('leaving only refreshes the recap on boot scope without a view', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('leaving', addr)], noViewBoot)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr]
		});
	});

	test('two changes on the same address produce one recap refresh', () => {
		const addr = taskAddr();
		const plan = planBoardRefresh([change('stay', addr), change('advance', addr)], viewable);
		expect(plan.recapRefresh).toHaveLength(1);
		expect(plan.recapRefresh[0]).toBe(addr);
	});

	test('stay plus leaving keeps the timeline reload and drops worklistOnly', () => {
		const stay = taskAddr('register');
		const leaving = taskAddr('seal');
		const plan = planBoardRefresh(
			[change('leaving', leaving), change('stay', stay)],
			noViewInstance
		);
		expect(plan.timeline).toBe(true);
		expect(plan.worklistOnly).toBe(false);
		expect(plan.recapRefresh).toHaveLength(2);
	});
});
