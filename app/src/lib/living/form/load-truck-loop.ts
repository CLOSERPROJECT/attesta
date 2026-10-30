import { parse as parseYaml } from 'yaml';
import { parseAuthoredPayload } from './dynamic-properties.js';

export type JobTemplate = {
	repeatable: boolean;
	instanceId: string;
	label: string;
	startLabel?: string;
	startsWhen?: unknown;
	tasks: Record<string, unknown>;
};

export type TruckLoopConfig = {
	id: string;
	org?: string;
	version?: string;
	jobs: Record<string, JobTemplate>;
};

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Parse truck-loop fixture YAML (steps/substeps) → living jobs/tasks for FormPlan tests.
 */
export function loadTruckLoop(yamlText: string): TruckLoopConfig {
	const doc = parseYaml(yamlText);
	if (!isPlainObject(doc)) throw new Error('truck-loop YAML must be a mapping');
	if (typeof doc.id !== 'string' || doc.id.length === 0) {
		throw new Error('truck-loop YAML id required');
	}
	const steps = isPlainObject(doc.steps) ? doc.steps : null;
	if (!steps) throw new Error('truck-loop YAML requires steps');

	const jobs: Record<string, JobTemplate> = {};
	for (const [jobId, raw] of Object.entries(steps)) {
		if (!isPlainObject(raw)) throw new Error(`steps.${jobId} must be a mapping`);
		const substeps = isPlainObject(raw.substeps) ? raw.substeps : null;
		if (!substeps) throw new Error(`steps.${jobId}.substeps required`);
		const tasks: Record<string, unknown> = {};
		for (const [taskId, taskRaw] of Object.entries(substeps)) {
			tasks[taskId] = parseAuthoredPayload(`jobs.${jobId}.tasks.${taskId}`, taskRaw);
		}
		const job: JobTemplate = {
			repeatable: raw.repeatable === true,
			instanceId: typeof raw.instanceId === 'string' ? raw.instanceId : '',
			label: typeof raw.label === 'string' ? raw.label : jobId,
			tasks
		};
		if (typeof raw.startLabel === 'string' && raw.startLabel.length > 0) {
			job.startLabel = raw.startLabel;
		}
		if (raw.startsWhen !== undefined) job.startsWhen = raw.startsWhen;
		jobs[jobId] = job;
	}

	return {
		id: doc.id,
		...(typeof doc.org === 'string' ? { org: doc.org } : {}),
		jobs
	};
}

export function taskConfig(
	job: JobTemplate | undefined,
	taskId: string
): Record<string, unknown> | null {
	if (!job) return null;
	const raw = job.tasks[taskId];
	return isPlainObject(raw) ? raw : null;
}

/** Experiment alias kept for ported tests. */
export const substepConfig = taskConfig;
