import {
	listEntities,
	type DomainSnapshot,
	type EntityType
} from './domain-snapshot.js';
import { evaluateSafe, formatJsonataError, type JsonataBindings } from './jsonata-safe.js';

export type ReadDef = {
	entity: EntityType;
	where?: string;
	key: string;
};

export type ResolveReadsError =
	| { kind: 'invalid-read'; message: string }
	| { kind: 'jsonata'; message: string };

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isEntityType(value: unknown): value is EntityType {
	return value === 'load' || value === 'pallet' || value === 'container';
}

export function parseReadDef(
	raw: unknown
): { ok: true; value: ReadDef } | { ok: false; error: ResolveReadsError } {
	if (!isPlainObject(raw)) {
		return { ok: false, error: { kind: 'invalid-read', message: 'read must be a mapping' } };
	}
	if (!isEntityType(raw.entity)) {
		return {
			ok: false,
			error: {
				kind: 'invalid-read',
				message: 'read.entity must be load, pallet, or container'
			}
		};
	}
	if (typeof raw.key !== 'string' || raw.key.length === 0) {
		return {
			ok: false,
			error: { kind: 'invalid-read', message: 'read.key must be a non-empty string' }
		};
	}
	if (raw.where !== undefined && typeof raw.where !== 'string') {
		return {
			ok: false,
			error: { kind: 'invalid-read', message: 'read.where must be a string when present' }
		};
	}
	return {
		ok: true,
		value: {
			entity: raw.entity,
			where: typeof raw.where === 'string' ? raw.where : undefined,
			key: raw.key
		}
	};
}

export type ResolveReadOptions = {
	/** Active tab’s Step instance for `$instance`. */
	instance?: { id: string; label: string };
};

function domainBindings(
	snapshot: DomainSnapshot,
	options: ResolveReadOptions = {}
): JsonataBindings {
	const bindings: JsonataBindings = {
		domain: {
			load: snapshot.load,
			pallet: snapshot.pallet,
			container: snapshot.container,
			submissions: snapshot.submissions
		}
	};
	if (options.instance) {
		bindings.instance = options.instance;
	}
	return bindings;
}

/**
 * `$` = entity facts; `key: id` uses the store map key.
 * Result order follows entity map insertion order.
 */
export async function resolveRead(
	snapshot: DomainSnapshot,
	read: ReadDef,
	options: ResolveReadOptions = {}
): Promise<{ ok: true; ids: string[] } | { ok: false; error: ResolveReadsError }> {
	const entities = listEntities(snapshot, read.entity);
	const ids: string[] = [];
	const bindings = domainBindings(snapshot, options);

	for (const [id, facts] of Object.entries(entities)) {
		if (read.where) {
			const result = await evaluateSafe(read.where, facts, bindings);
			if (!result.ok) {
				return { ok: false, error: { kind: 'jsonata', message: formatJsonataError(result.error) } };
			}
			if (!result.value) continue;
		}

		if (read.key === 'id') {
			ids.push(id);
			continue;
		}

		const keyed = facts[read.key];
		if (typeof keyed === 'string' && keyed.length > 0) {
			ids.push(keyed);
		}
	}

	return { ok: true, ids };
}

export function formatResolveReadsError(error: ResolveReadsError): string {
	if (error.kind === 'jsonata') return error.message;
	return `${error.kind}: ${error.message}`;
}
