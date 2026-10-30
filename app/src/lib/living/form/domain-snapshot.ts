/** Living Domain slice FormPlan / resolve-reads need (entities + submissions). */
export type EntityType = 'load' | 'pallet' | 'container';

export type EntityFacts = Record<string, unknown>;

export type EntityMap = Record<string, EntityFacts>;

export type Submission = {
	jobId?: string;
	taskId?: string;
	/** Experiment legacy keys still present on some fixtures/tests. */
	stepId?: string;
	substepId?: string;
	jobInstanceId?: string;
	stepInstanceId?: string;
	timestamp?: string;
	userId?: string;
	facts: Record<string, unknown>;
	idsTouched?: string[];
};

export type DomainSnapshot = {
	load: EntityMap;
	pallet: EntityMap;
	container: EntityMap;
	submissions: Submission[];
	jobInstances?: unknown[];
};

export function listEntities(snapshot: DomainSnapshot, entity: EntityType): EntityMap {
	return snapshot[entity];
}

export function emptySnapshot(): DomainSnapshot {
	return { load: {}, pallet: {}, container: {}, submissions: [], jobInstances: [] };
}
