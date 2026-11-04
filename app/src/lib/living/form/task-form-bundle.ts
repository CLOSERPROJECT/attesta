import type { Schema, UiSchemaRoot } from '@sjsf/form';
import type { ChromeEntry, TaskFormPlan } from './task-form-plan.js';
import type { DerivedKeyEntry } from './derived-keys.js';

/** Wire Task form bundle from living Load/Commit. */
export type TaskFormBundle = {
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	/** Nested leaf under a Task group; omitted for flat Tasks. */
	groupId?: string;
	/** Empty when provisional spawn (mirror empty jobInstanceId for Job start). */
	groupInstanceId?: string;
	title: string;
	surface: 'live';
	schema: Schema;
	uiSchema: UiSchemaRoot;
	initialData: Record<string, unknown>;
	rowIds?: string[];
	columns?: { id: string; value: string };
	chrome: ChromeEntry[];
	derived?: Record<string, DerivedKeyEntry>;
	chromeFieldIds: string[];
	sealFieldIds: string[];
	projectionKey?: string;
	bindings: Record<string, unknown>;
};

export function bundleMatches(
	bundle: TaskFormBundle | null | undefined,
	jobId: string,
	taskId: string,
	jobInstanceId: string,
	groupId = '',
	groupInstanceId = ''
): bundle is TaskFormBundle {
	return (
		bundle != null &&
		bundle.jobId === jobId &&
		bundle.taskId === taskId &&
		bundle.jobInstanceId === jobInstanceId &&
		(bundle.groupId ?? '') === groupId &&
		(bundle.groupInstanceId ?? '') === groupInstanceId
	);
}

/** Lookup one mount payload from the Load/Commit bundle set. */
export function findTaskFormBundle(
	bundles: readonly TaskFormBundle[] | null | undefined,
	jobId: string,
	taskId: string,
	jobInstanceId: string,
	groupId = '',
	groupInstanceId = ''
): TaskFormBundle | null {
	if (!bundles?.length) return null;
	return (
		bundles.find((b) =>
			bundleMatches(b, jobId, taskId, jobInstanceId, groupId, groupInstanceId)
		) ?? null
	);
}

/** Frozen FormPlan from a server Task form bundle — no authored+Domain compile. */
export function planFromTaskFormBundle(bundle: TaskFormBundle): TaskFormPlan {
	return {
		surface: bundle.surface,
		schemas: {
			schema: bundle.schema,
			uiSchema: bundle.uiSchema
		},
		...(bundle.derived !== undefined ? { derived: bundle.derived } : {}),
		chrome: bundle.chrome ?? [],
		chromeFieldIds: bundle.chromeFieldIds ?? [],
		projectionKey: bundle.projectionKey ?? '',
		// Chrome strip + cell omit live in Go prep; SPA seal is pass-through.
		sealSubmission: (value) => value
	};
}

/**
 * Domain bag from Task form bundle bindings for live Chrome/Derived JSONata —
 * no SPA Domain/Config snapshot.
 */
export function domainFromBundleBindings(bindings: Record<string, unknown>): unknown {
	return bindings.domain;
}

export function catalogsFromBundleBindings(bindings: Record<string, unknown>): unknown {
	return bindings.catalogs;
}

/** Server-precomputed dataReport displays (`bindings.chromeDisplay`). */
export function chromeDisplayFromBundleBindings(
	bindings: Record<string, unknown>
): Record<string, unknown> | undefined {
	const raw = bindings.chromeDisplay;
	if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return undefined;
	return raw as Record<string, unknown>;
}
