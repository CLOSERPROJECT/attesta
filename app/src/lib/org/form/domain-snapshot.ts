/** Org domain slice for chrome / Derived bindings (dynamic kinds + submissions). */

export type EntityFacts = Record<string, unknown>;

export type EntityMap = Record<string, EntityFacts>;

export type EntityTouch = {
	op: 'mint' | 'patch';
	entity: string;
	id: string;
};

export type Submission = {
	jobId?: string;
	taskId?: string;
	jobInstanceId?: string;
	timestamp?: string;
	userId?: string;
	facts: Record<string, unknown>;
	entitiesTouched?: EntityTouch[];
};

/** Server bag: kind names are Org Config vocabulary (e.g. lot / pallet / container). */
export type DomainSnapshot = {
	submissions: Submission[];
	jobInstances?: unknown[];
} & Record<string, EntityMap | Submission[] | unknown[] | undefined>;

export function listEntities(snapshot: DomainSnapshot, entity: string): EntityMap {
	const value = snapshot[entity];
	if (value == null || Array.isArray(value) || typeof value !== 'object') {
		return {};
	}
	return value as EntityMap;
}

export function emptySnapshot(): DomainSnapshot {
	return { submissions: [], jobInstances: [] };
}
