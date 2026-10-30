import { getContext, hasContext, setContext } from 'svelte';

export type LeftoverContext = {
	ids: string[];
	columns: { id: string; value: string };
	valueType: 'number' | 'progress';
};

const KEY = Symbol('attesta-leftover-context');

export function setLeftoverContext(ctx: LeftoverContext): void {
	setContext(KEY, ctx);
}

export function getLeftoverContext(): LeftoverContext {
	if (!hasContext(KEY)) {
		throw new Error('leftover context missing — mount TableFormShell first');
	}
	return getContext(KEY);
}
