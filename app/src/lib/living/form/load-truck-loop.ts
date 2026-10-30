import { parse as parseYaml } from 'yaml';
import { parseAuthoredPayload } from './dynamic-properties.js';
import type { TaskConfig } from '../types.js';
import { taskById } from '../types.js';

export type JobTemplate = {
	repeatable: boolean;
	instanceId: string;
	label: string;
	startLabel?: string;
	startsWhen?: unknown;
	tasks: TaskConfig[];
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
		if (!Array.isArray(raw.substeps) || raw.substeps.length === 0) {
			throw new Error(`steps.${jobId}.substeps must be a non-empty list`);
		}
		const tasks: TaskConfig[] = [];
		const seen = new Set<string>();
		for (let i = 0; i < raw.substeps.length; i++) {
			const taskRaw = raw.substeps[i];
			if (!isPlainObject(taskRaw) || typeof taskRaw.id !== 'string' || taskRaw.id.length === 0) {
				throw new Error(`steps.${jobId}.substeps[${i}]: id required`);
			}
			const id = taskRaw.id;
			if (seen.has(id)) {
				throw new Error(`steps.${jobId}.substeps: duplicate id ${id}`);
			}
			seen.add(id);
			const payload = parseAuthoredPayload(`jobs.${jobId}.tasks.${id}`, taskRaw);
			tasks.push({ ...payload, id });
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
	const raw = taskById(job.tasks, taskId);
	return raw ?? null;
}

/** Experiment alias kept for ported tests. */
export const substepConfig = taskConfig;
