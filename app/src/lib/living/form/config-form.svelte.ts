import type { Schema, UiSchemaRoot } from '@sjsf/form';
import { untrack } from 'svelte';
import { getAtPath, setAtPath } from './derived-keys.js';
import { evaluateSafe } from './jsonata-safe.js';

export type ConfigFormDerivedEntry = {
	expr: string;
	/** ADR-0016: default false — operator cannot keep a dirty value. */
	overridable?: boolean;
};

export type ConfigFormOptions<T extends Record<string, unknown> = Record<string, unknown>> = {
	schema: Schema;
	uiSchema?: UiSchemaRoot;
	initialValue: T;
	derived: Record<string, ConfigFormDerivedEntry>;
	/**
	 * Host injects later (reads / domain snapshot). Bound as `$domain` beside `$data`
	 * (= live form value) when evaluating derived exprs — Attesta has no kernel `$refs`.
	 */
	getDomain?: () => unknown;
};

type NormalizedDerived = {
	expr: string;
	overridable: boolean;
};

/**
 * Local derived + per-path override (ADR-0016 policy, no kernel).
 * Pair with `createConfigForm` via `valueBinding` — Bun unit tests cannot load
 * sjsf `createForm` subpaths the way Vite does.
 */
export class ConfigForm<T extends Record<string, unknown> = Record<string, unknown>> {
	readonly schema: Schema;
	readonly uiSchema: UiSchemaRoot;
	#derived: Record<string, NormalizedDerived>;
	#getDomain: () => unknown;
	#overrides = new Set<string>();
	#overrideEpoch = $state(0);
	/** $state so derivedLeaf can track computed() after async apply. */
	#computed = $state(new Map<string, unknown>());
	#generation = 0;
	#applying = false;
	#pending: Promise<void> = Promise.resolve();
	value = $state({} as T);

	constructor(options: ConfigFormOptions<T>) {
		this.schema = options.schema;
		this.uiSchema = options.uiSchema ?? {};
		this.#derived = normalizeDerived(options.derived);
		this.#getDomain = options.getDomain ?? (() => undefined);
		this.value = $state.snapshot(options.initialValue) as T;

		watchDerivedApply({
			getValue: () => this.value,
			getOverrideEpoch: () => this.#overrideEpoch,
			schedule: () => {
				this.#scheduleApply();
			}
		});
		this.#scheduleApply();
	}

	/** Await in-flight derived re-eval (tests + hosts that need a settled snapshot). */
	whenSettled = (): Promise<void> => this.#pending;

	/** sjsf `createForm` value tuple — setter runs override / snap-back policy. */
	valueBinding = (): [() => T, (next: T) => void] => [
		() => this.value,
		(next) => {
			this.#onOperatorValue(next);
		}
	];

	isOverridden = (path: string): boolean => {
		void this.#overrideEpoch;
		return this.#overrides.has(path);
	};

	isOverridable = (path: string): boolean => this.#derived[path]?.overridable === true;

	/** Last successful expr result for path (undefined until first settle). */
	computed = (path: string): unknown => this.#computed.get(path);

	/**
	 * Shape for `setConfigFormContext` — same paths as `derived` keys.
	 */
	derivedContext = (): {
		isOverridden: (path: string) => boolean;
		isOverridable: (path: string) => boolean;
		computed: (path: string) => unknown;
		restore: (path: string) => void;
		writePath: (path: string, value: unknown) => void;
	} => ({
		isOverridden: this.isOverridden,
		isOverridable: this.isOverridable,
		computed: this.computed,
		restore: this.restore,
		writePath: this.writePath
	});

	restore = (path: string): void => {
		if (!this.#overrides.has(path)) return;
		this.#overrides = new Set(this.#overrides);
		this.#overrides.delete(path);
		this.#overrideEpoch += 1;
	};

	restoreAll = (): void => {
		if (this.#overrides.size === 0) return;
		this.#overrides = new Set();
		this.#overrideEpoch += 1;
	};

	/**
	 * Sole operator write for derivedLeaf override paths (ADR-0016).
	 * Overridable derived → override set; non-overridable → ignored (effect keeps expr).
	 */
	writePath = (path: string, value: unknown): void => {
		const entry = this.#derived[path];
		if (entry && !entry.overridable) {
			return;
		}
		if (entry?.overridable && !this.#overrides.has(path)) {
			this.#overrides = new Set(this.#overrides);
			this.#overrides.add(path);
			this.#overrideEpoch += 1;
		}
		this.#applying = true;
		this.value = setAtPath(this.value as Record<string, unknown>, path, value) as T;
		this.#applying = false;
	};

	#onOperatorValue = (next: T): void => {
		if (this.#applying) {
			this.value = next;
			return;
		}
		const prev = untrack(() => this.value);
		let merged = next as Record<string, unknown>;
		let overridesChanged = false;
		const nextOverrides = new Set(this.#overrides);

		for (const [path, entry] of Object.entries(this.#derived)) {
			const nextAt = getAtPath(next, path);
			const prevAt = getAtPath(prev, path);
			const computed = this.#computed.get(path);
			if (!entry.overridable) {
				if (!Object.is(nextAt, computed)) {
					merged = setAtPath(merged, path, computed);
				}
				continue;
			}
			if (nextOverrides.has(path)) continue;
			if (!Object.is(nextAt, prevAt) && !Object.is(nextAt, computed)) {
				nextOverrides.add(path);
				overridesChanged = true;
			}
		}

		this.#applying = true;
		if (overridesChanged) {
			this.#overrides = nextOverrides;
			this.#overrideEpoch += 1;
		}
		this.value = merged as T;
		this.#applying = false;
	};

	#scheduleApply = (): void => {
		const generation = ++this.#generation;
		this.#pending = this.#applyDerived(generation);
	};

	#applyDerived = async (generation: number): Promise<void> => {
		const data = untrack(() => $state.snapshot(this.value));
		const domain = untrack(() => this.#getDomain());
		const overrides = untrack(() => new Set(this.#overrides));
		const computed = new Map<string, unknown>();
		let next = data as Record<string, unknown>;

		for (const [path, entry] of Object.entries(this.#derived)) {
			const result = await evaluateSafe(entry.expr, {}, { data, domain });
			if (generation !== this.#generation) return;
			if (!result.ok) continue;
			computed.set(path, result.value);
			if (overrides.has(path)) continue;
			if (!Object.is(getAtPath(next, path), result.value)) {
				next = setAtPath(next, path, result.value);
			}
		}

		if (generation !== this.#generation) return;
		this.#computed = computed;
		if (JSON.stringify(next) === JSON.stringify(data)) return;
		this.#applying = true;
		this.value = next as T;
		this.#applying = false;
	};
}

/** Why: `$effect.root` beside the class matches deep-local-draft-sync (testable under Bun). */
function watchDerivedApply(args: {
	getValue: () => unknown;
	getOverrideEpoch: () => number;
	schedule: () => void;
}): void {
	$effect.root(() => {
		$effect(() => {
			void JSON.stringify(args.getValue());
			void args.getOverrideEpoch();
			args.schedule();
		});
	});
}

function normalizeDerived(
	derived: Record<string, ConfigFormDerivedEntry>
): Record<string, NormalizedDerived> {
	const out: Record<string, NormalizedDerived> = {};
	for (const [path, entry] of Object.entries(derived)) {
		out[path] = {
			expr: entry.expr,
			overridable: entry.overridable === true
		};
	}
	return out;
}
