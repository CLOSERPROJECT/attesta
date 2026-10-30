import { taskById, taskMayBeUnlockGate } from './unlock.js';
import type { Domain, JobConfig, OrgConfig, Submission } from './types.js';

export type StartsWhenGate = {
	task: { jobId: string; taskId: string };
	quantifier: string;
};

/** Resolve the Task config referenced by a startsWhen gate (for completeness checks). */
export function referencedStartsWhenTask(
	config: OrgConfig | null | undefined,
	startsWhen: StartsWhenGate | null | undefined
): unknown {
	if (!config?.jobs || !startsWhen?.task) return null;
	const job = config.jobs[startsWhen.task.jobId];
	if (!job?.tasks) return null;
	return taskById(job.tasks, startsWhen.task.taskId) ?? null;
}

/**
 * Absent gate always allows. `some` = ≥1 matching Submission; `all` = every
 * Job instance of the referenced Job has that Task Submission (and ≥1 exists).
 * Completeness `some` Tasks never satisfy (fail closed).
 */
export function isStartsWhenSatisfied(
	startsWhen: StartsWhenGate | null | undefined,
	submissions: Submission[],
	jobInstances: Domain['jobInstances'],
	referencedTask: unknown
): boolean {
	if (!startsWhen) return true;
	if (referencedTask == null || !taskMayBeUnlockGate(referencedTask)) return false;

	const { jobId, taskId } = startsWhen.task;
	const matching = submissions.filter((s) => s.jobId === jobId && s.taskId === taskId);

	if (startsWhen.quantifier === 'all') {
		const instances = jobInstances.filter((inst) => inst.jobId === jobId);
		if (instances.length === 0) return false;
		return instances.every((inst) =>
			matching.some((s) => s.jobInstanceId === inst.id)
		);
	}

	return matching.length > 0;
}

/** Start affordance when startsWhen holds and (repeatable or no instance yet). */
export function canStartJob(args: {
	job: JobConfig;
	jobId: string;
	domain: Domain;
	config: OrgConfig | null | undefined;
}): boolean {
	const { job, jobId, domain, config } = args;
	const n = domain.jobInstances.filter((inst) => inst.jobId === jobId).length;
	if (!(job.repeatable || n === 0)) return false;
	const gate = job.startsWhen ?? null;
	return isStartsWhenSatisfied(
		gate,
		domain.submissions,
		domain.jobInstances,
		referencedStartsWhenTask(config, gate)
	);
}
