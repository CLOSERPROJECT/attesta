import { getContext, hasContext, setContext } from 'svelte';

/**
 * Override/restore surface published by ConfigForm and consumed by
 * derivedLeafField. Paths are concrete Derived keys from
 * `derived-keys.ts` (dotted, or JSON Pointer when a segment contains `.`).
 *
 * Map a ConfigForm instance when mounting:
 * `setConfigFormContext({ isOverridden: form.isOverridden, … writePath: form.writePath })`
 * plus `isOverridable` / `computed` from the derived map / `#computed`
 * (host adapts until the class widens).
 */
export type ConfigFormDerivedContext = {
	isOverridden(path: string): boolean;
	isOverridable(path: string): boolean;
	/** Live expr result for path (display when not overridden; restore target). */
	computed(path: string): unknown;
	restore(path: string): void;
	/** Operator write — matches ConfigForm.writePath (marks override when allowed). */
	writePath(path: string, value: unknown): void;
};

const KEY = Symbol('attesta-config-form');

export function setConfigFormContext(ctx: ConfigFormDerivedContext): void {
	setContext(KEY, ctx);
}

export function getConfigFormContext(): ConfigFormDerivedContext {
	if (!hasContext(KEY)) {
		throw new Error('config form context missing — mount ConfigForm host first');
	}
	return getContext(KEY);
}

export function tryGetConfigFormContext(): ConfigFormDerivedContext | null {
	return hasContext(KEY) ? getContext(KEY) : null;
}
