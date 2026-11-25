import { describe, expect, test, vi } from 'vitest';

import type { TaskAddress } from '../task-address.js';
import type { TimelineSubstep, TimelineView } from '../timeline/types.js';
import { BoardEmbedOpen } from './board-embed-open.svelte.js';
import { BoardJustCompleted } from './board-just-completed.svelte.js';
import {
	BoardTimelineModel,
	addressVisibleOnTimeline,
	projectVisibleSteps,
	timelineSubstepShown
} from './board-timeline-model.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

const base = {
	stream: 'pv-recovery',
	instance: '261102-1',
	label: 'Camion',
	status: 'active'
} as const;

const stream = base.stream;
const instance = base.instance;

function lotSub(
	lotId: string,
	opts: { orgActionable?: boolean; status?: string } = {}
): TimelineSubstep {
	return {
		substepId: 'lot_processing.weigh',
		title: 'Pesa',
		status: opts.status ?? (opts.orgActionable ? 'available' : 'locked'),
		orgActionable: opts.orgActionable ?? false,
		href: `/app/${stream}/i/${instance}/lot_processing/${lotId}/weigh`
	};
}

function multiLotView(): TimelineView {
	return {
		...base,
		steps: [
			{
				stepId: '2',
				title: 'Lots',
				substepCount: 2,
				expanded: true,
				rows: [
					{
						id: 'lot_processing.DA-1',
						label: 'DA-1',
						expanded: true,
						substeps: [lotSub('DA-1', { orgActionable: true })]
					},
					{
						id: 'lot_processing.CO-1',
						label: 'CO-1',
						expanded: true,
						substeps: [lotSub('CO-1', { orgActionable: false })]
					}
				]
			}
		]
	};
}

function soleActionableView(): TimelineView {
	return {
		...base,
		steps: [
			{
				stepId: '1',
				title: 'Intake',
				substepCount: 1,
				expanded: true,
				substeps: [
					{
						substepId: 'intake.register',
						title: 'Register',
						status: 'available',
						orgActionable: true,
						href: `/app/${stream}/i/${instance}/intake/${instance}/register`
					}
				]
			}
		]
	};
}

const focusDA: BoardTaskFocus = {
	jobId: 'lot_processing',
	jobInstanceId: 'DA-1',
	taskId: 'weigh'
};

const addrDA = (): TaskAddress => ({
	stream,
	instance,
	jobId: 'lot_processing',
	jobInstanceId: 'DA-1',
	taskId: 'weigh',
	provisionalJob: false
});

describe('BoardTimelineModel visibility', () => {
	test('multi actionable + no selection → only actionable; locked sibling lot hidden', () => {
		const view = multiLotView();
		// Add a second actionable so there is no sole auto-select
		view.steps[0]!.rows![1]!.substeps = [
			lotSub('CO-1', { orgActionable: true })
		];

		const projected = projectVisibleSteps(view, {
			showAll: false,
			focus: null,
			stream,
			instance
		});
		expect(projected).toHaveLength(1);
		expect(projected[0]!.rows).toHaveLength(2);
		expect(
			projected[0]!.rows!.every((r) =>
				r.substeps.every((s) => s.sub.orgActionable)
			)
		).toBe(true);

		const lockedSibling = multiLotView();
		const lockedOnly = projectVisibleSteps(lockedSibling, {
			showAll: false,
			focus: null,
			stream,
			instance
		});
		expect(lockedOnly[0]!.rows).toHaveLength(1);
		expect(lockedOnly[0]!.rows![0]!.row.id).toBe('lot_processing.DA-1');
	});

	test('select DA weigh → that instance visible; CO sibling not', () => {
		const view = multiLotView();
		// Make DA non-actionable so visibility depends on selection
		view.steps[0]!.rows![0]!.substeps = [
			lotSub('DA-1', { orgActionable: false })
		];

		expect(
			timelineSubstepShown(view.steps[0]!.rows![0]!.substeps[0]!, {
				showAll: false,
				focus: focusDA,
				stream,
				instance
			})
		).toBe(true);
		expect(
			timelineSubstepShown(view.steps[0]!.rows![1]!.substeps[0]!, {
				showAll: false,
				focus: focusDA,
				stream,
				instance
			})
		).toBe(false);

		const projected = projectVisibleSteps(view, {
			showAll: false,
			focus: focusDA,
			stream,
			instance
		});
		expect(projected[0]!.rows).toHaveLength(1);
		expect(projected[0]!.rows![0]!.row.id).toBe('lot_processing.DA-1');
		expect(projected[0]!.rows![0]!.mark).toBe('2A');
	});

	test('sole actionable → selection + body open after applyLoadPolicy', () => {
		const syncUrl = vi.fn();
		const embed = new BoardEmbedOpen();
		const model = new BoardTimelineModel({ syncUrl, embedOpen: embed });
		model.setScope(stream, instance);
		model.setView(soleActionableView());

		expect(model.selection).toBeNull();
		model.applyLoadPolicy();

		expect(model.selection?.taskId).toBe('register');
		expect(syncUrl).toHaveBeenCalled();
		const sole = model.soleActionable!;
		expect(model.isBodyOpen(sole)).toBe(true);
		expect(model.isSelected(sole)).toBe(true);
	});

	test('address visible is instance-scoped', () => {
		const view = multiLotView();
		expect(addressVisibleOnTimeline(view, addrDA())).toBe(true);
		expect(
			addressVisibleOnTimeline(view, {
				...addrDA(),
				jobInstanceId: 'CO-1'
			})
		).toBe(true);
		expect(
			addressVisibleOnTimeline(view, {
				...addrDA(),
				jobInstanceId: 'XX-9'
			})
		).toBe(false);

		const syncUrl = vi.fn();
		const model = new BoardTimelineModel({
			syncUrl,
			embedOpen: new BoardEmbedOpen()
		});
		model.setScope(stream, instance);
		model.setView(view);
		expect(model.addressVisible(addrDA())).toBe(true);
		expect(
			model.addressVisible({ ...addrDA(), jobInstanceId: 'XX-9' })
		).toBe(false);
	});

	test('many actionables → applyLoadPolicy leaves selection empty', () => {
		const view = multiLotView();
		view.steps[0]!.rows![1]!.substeps = [
			lotSub('CO-1', { orgActionable: true })
		];
		const syncUrl = vi.fn();
		const model = new BoardTimelineModel({
			syncUrl,
			embedOpen: new BoardEmbedOpen()
		});
		model.setScope(stream, instance);
		model.setView(view);
		model.applyLoadPolicy();
		expect(model.selection).toBeNull();
		expect(syncUrl).not.toHaveBeenCalled();
		expect(model.isEmpty).toBe(false);
		expect(model.visibleSteps[0]!.rows).toHaveLength(2);
	});

	test('setFocusFromUrl selects and opens body', () => {
		const syncUrl = vi.fn();
		const embed = new BoardEmbedOpen();
		const model = new BoardTimelineModel({ syncUrl, embedOpen: embed });
		model.setScope(stream, instance);
		model.setView(multiLotView());
		model.setFocusFromUrl(focusDA);
		expect(model.selection).toEqual(focusDA);
		expect(model.isBodyOpen(addrDA())).toBe(true);
	});

	test('setFocusFromUrl same focus does not reopen after user collapse', () => {
		const syncUrl = vi.fn();
		const embed = new BoardEmbedOpen();
		const model = new BoardTimelineModel({ syncUrl, embedOpen: embed });
		model.setScope(stream, instance);
		model.setView(multiLotView());
		model.setFocusFromUrl(focusDA);
		expect(model.isBodyOpen(addrDA())).toBe(true);
		model.setBodyOpen(addrDA(), false);
		expect(model.isBodyOpen(addrDA())).toBe(false);
		model.setFocusFromUrl({ ...focusDA });
		expect(model.isBodyOpen(addrDA())).toBe(false);
		expect(model.selection).toEqual(focusDA);
	});

	// Collapse-all: embeds via collapseAllBodies; non-embed details via nest epoch (not step/row nests).
	test('collapseAllBodies closes open embed panels', () => {
		const syncUrl = vi.fn();
		const embed = new BoardEmbedOpen();
		const model = new BoardTimelineModel({ syncUrl, embedOpen: embed });
		model.setScope(stream, instance);
		model.setView(multiLotView());
		model.setBodyOpen(addrDA(), true);
		expect(model.isBodyOpen(addrDA())).toBe(true);
		model.collapseAllBodies();
		expect(model.isBodyOpen(addrDA())).toBe(false);
	});
});

describe('BoardTimelineModel just-completed pin', () => {
	function doneOnlyView(): TimelineView {
		const view = multiLotView();
		view.steps[0]!.rows![0]!.substeps = [
			lotSub('DA-1', { orgActionable: false, status: 'done' })
		];
		view.steps[0]!.rows![1]!.substeps = [
			lotSub('CO-1', { orgActionable: false, status: 'locked' })
		];
		return view;
	}

	function modelWithPin(pin: BoardJustCompleted = new BoardJustCompleted()) {
		const syncUrl = vi.fn();
		const model = new BoardTimelineModel({
			syncUrl,
			embedOpen: new BoardEmbedOpen(),
			justCompleted: pin
		});
		model.setScope(stream, instance);
		model.setView(doneOnlyView());
		return { model, pin, syncUrl };
	}

	test('pin shows done under available-only; clear hides', () => {
		const { model } = modelWithPin();
		const done = lotSub('DA-1', { orgActionable: false, status: 'done' });

		expect(
			timelineSubstepShown(done, {
				showAll: false,
				stream,
				instance,
				justCompleted: null
			})
		).toBe(false);

		model.pinJustCompleted(addrDA(), 'sub-9');
		expect(model.visibleSteps[0]!.rows).toHaveLength(1);
		expect(model.visibleSteps[0]!.rows![0]!.row.id).toBe(
			'lot_processing.DA-1'
		);

		expect(
			timelineSubstepShown(done, {
				showAll: false,
				stream,
				instance,
				justCompleted: {
					has: (a) =>
						a.jobInstanceId === 'DA-1' && a.taskId === 'weigh'
				}
			})
		).toBe(true);

		model.clearJustCompleted();
		expect(model.visibleSteps).toHaveLength(0);
	});

	test('select / clearSelection / setFocusFromUrl clear pin when focus leaves', () => {
		const { model, pin } = modelWithPin();
		const addrCO = (): TaskAddress => ({
			...addrDA(),
			jobInstanceId: 'CO-1'
		});

		model.pinJustCompleted(addrDA());
		expect(model.visibleSteps[0]!.rows).toHaveLength(1);

		model.select(addrCO());
		expect(pin.get()).toBeNull();
		expect(model.visibleSteps[0]!.rows).toHaveLength(1);
		expect(model.visibleSteps[0]!.rows![0]!.row.id).toBe(
			'lot_processing.CO-1'
		);

		model.pinJustCompleted(addrDA());
		model.clearSelection();
		expect(pin.get()).toBeNull();
		expect(model.visibleSteps).toHaveLength(0);

		model.pinJustCompleted(addrDA());
		model.setFocusFromUrl({
			jobId: 'lot_processing',
			jobInstanceId: 'CO-1',
			taskId: 'weigh'
		});
		expect(pin.get()).toBeNull();
		expect(model.visibleSteps[0]!.rows![0]!.row.id).toBe(
			'lot_processing.CO-1'
		);
	});

	test('select same pinned address keeps pin', () => {
		const { model } = modelWithPin();
		model.pinJustCompleted(addrDA());
		model.select(addrDA());
		expect(model.visibleSteps[0]!.rows).toHaveLength(1);
	});

	test('showAll still shows everything; pin is irrelevant', () => {
		const { model } = modelWithPin();
		model.setShowAll(true);
		expect(model.visibleSteps[0]!.rows).toHaveLength(2);
		model.pinJustCompleted(addrDA());
		expect(model.visibleSteps[0]!.rows).toHaveLength(2);
	});

	test('two models sharing store retain pin across remount', () => {
		const shared = new BoardJustCompleted();
		const first = modelWithPin(shared);
		first.model.pinJustCompleted(addrDA(), 'sub-remount');

		const second = modelWithPin(shared);
		expect(second.model.visibleSteps[0]!.rows).toHaveLength(1);
		expect(shared.get()?.submissionId).toBe('sub-remount');
	});

	test('isEmpty is false when only a pinned done card is visible', () => {
		const { model } = modelWithPin();
		expect(model.isEmpty).toBe(true);
		model.pinJustCompleted(addrDA());
		expect(model.isEmpty).toBe(false);
		expect(model.visibleSteps).toHaveLength(1);
	});
});
