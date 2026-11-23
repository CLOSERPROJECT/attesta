import { describe, expect, test, vi } from 'vitest';

import type { TaskAddress } from '../task-address.js';
import type { TimelineSubstep, TimelineView } from '../timeline/types.js';
import { BoardEmbedOpen } from './board-embed-open.svelte.js';
import {
	BoardTimelineModel,
	addressVisibleOnTimeline,
	projectVisibleSteps,
	timelineSubstepShown
} from './board-timeline-model.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

const base = {
	stream: 'recycling',
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
