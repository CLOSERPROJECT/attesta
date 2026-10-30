import {
	dynamicPropertyKeys,
	isMapTableUiField,
	primaryDynamicProperty
} from './dynamic-properties.js';
import type { TaskConfig } from '../types.js';
import { orderedTaskIds, taskById } from '../unlock.js';

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Flat leftoverTable chrome (`ui:field: leftoverTable`) — TableFormShell path.
 * mapTable composition uses ConfigForm live-form instead.
 */
export function isLeftoverTableTask(substep: unknown): boolean {
	if (!isPlainObject(substep)) return false;
	const keys = dynamicPropertyKeys(substep);
	if (keys.length === 0) return false;
	const primary = primaryDynamicProperty(substep);
	if (primary === null) return false;
	return !isMapTableUiField(substep.uiSchema, primary);
}

/** Object-row mapTable + dynamicProperties — project leftovers, mount ConfigForm. */
export function isMapTableTask(substep: unknown): boolean {
	if (!isPlainObject(substep)) return false;
	const primary = primaryDynamicProperty(substep);
	if (primary === null) return false;
	return isMapTableUiField(substep.uiSchema, primary);
}

/** Leftover id projection (flat leftoverTable or mapTable). */
export function needsLeftoverProjection(substep: unknown): boolean {
	return isLeftoverTableTask(substep) || isMapTableTask(substep);
}

/** Authored `enumReads` — projectFormSchema injects string-field `enum` from reads. */
export function hasEnumReads(substep: unknown): boolean {
	if (!isPlainObject(substep)) return false;
	const enumReads = substep.enumReads;
	if (!isPlainObject(enumReads)) return false;
	return Object.keys(enumReads).length > 0;
}

/**
 * Any Substep whose live FormPlan must use projected schema/ui (leftovers and/or enums).
 * Keep distinct from {@link needsLeftoverProjection}: submit prep is leftover-only.
 */
export function needsSchemaProjection(substep: unknown): boolean {
	return needsLeftoverProjection(substep) || hasEnumReads(substep);
}

export function leftoverTableIdsFromJob(job: { tasks?: TaskConfig[] }): string[] {
	const tasks = job.tasks ?? [];
	return orderedTaskIds(tasks).filter((id) => isLeftoverTableTask(taskById(tasks, id)));
}

/** Tasks refreshed at open: leftoverTable, mapTable, and enumReads forms. */
export function leftoverProjectionIdsFromJob(job: { tasks?: TaskConfig[] }): string[] {
	const tasks = job.tasks ?? [];
	return orderedTaskIds(tasks).filter((id) => needsSchemaProjection(taskById(tasks, id)));
}
