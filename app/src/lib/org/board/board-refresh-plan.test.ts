import { describe, expect, test } from 'vitest';

import { normalizeTaskAddress, taskAddressKey, type TaskAddress } from '../task-address.js';
import {
	planBoardRefresh,
	type BoardRefreshFacts,
	type OwnDomainChange
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
	follow: OwnDomainChange['follow'],
	addr: TaskAddress,
	submitMany = false
): OwnDomainChange {
	return { origin: 'own', revision: 7, addr, follow, submitMany };
}

const ownDefaults = {
	revalidateForms: false,
	holdExempt: [] as string[],
	revalidateRecaps: true,
	background: false
};

const idleDefaults = {
	revalidateForms: false,
	holdExempt: [] as string[],
	revalidateRecaps: false,
	background: false
};

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
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('stay without submitMany keeps form bundles', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('stay', addr)], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('advance with a view reloads the timeline on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
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
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('advance without a view hydrates the Worklist on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], noViewInstance)).toEqual({
			timeline: false,
			worklistOnly: true,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('advance without a view does nothing extra on boot scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('advance', addr)], noViewBoot)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('leaving only refreshes the recap on instance scope', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('leaving', addr)], viewable)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
		});
	});

	test('leaving only refreshes the recap on boot scope without a view', () => {
		const addr = taskAddr();
		expect(planBoardRefresh([change('leaving', addr)], noViewBoot)).toEqual({
			timeline: false,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [addr],
			...ownDefaults
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

	test('foreign on instance scope revalidates forms and recaps in the background', () => {
		expect(planBoardRefresh([{ origin: 'foreign', revision: 2 }], viewable)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [],
			revalidateForms: true,
			holdExempt: [],
			revalidateRecaps: true,
			background: true
		});
	});

	test('foreign on boot scope only reloads the timeline', () => {
		expect(planBoardRefresh([{ origin: 'foreign', revision: 2 }], noViewBoot)).toEqual({
			timeline: true,
			worklistOnly: false,
			remountForms: false,
			recapRefresh: [],
			...idleDefaults
		});
	});

	test('own stay plus foreign holds the own key and turns both revalidations on', () => {
		const addr = taskAddr();
		const plan = planBoardRefresh(
			[change('stay', addr), { origin: 'foreign', revision: 2 }],
			viewable
		);
		expect(plan.timeline).toBe(true);
		expect(plan.remountForms).toBe(false);
		expect(plan.revalidateForms).toBe(true);
		expect(plan.revalidateRecaps).toBe(true);
		expect(plan.background).toBe(true);
		expect(plan.holdExempt).toEqual([taskAddressKey(addr)]);
		expect(plan.recapRefresh).toEqual([addr]);
	});

	test('own stay submitMany plus foreign remounts forms and skips revalidateForms', () => {
		const addr = taskAddr();
		const plan = planBoardRefresh(
			[change('stay', addr, true), { origin: 'foreign', revision: 2 }],
			viewable
		);
		expect(plan.remountForms).toBe(true);
		expect(plan.revalidateForms).toBe(false);
		expect(plan.holdExempt).toEqual([]);
		expect(plan.revalidateRecaps).toBe(true);
		expect(plan.background).toBe(true);
	});
});
