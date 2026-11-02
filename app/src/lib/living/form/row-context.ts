import { getContext, hasContext, setContext } from 'svelte';

export type RowContext = {
	ids: string[];
	columns: { id: string; value: string };
	valueType: 'number' | 'progress';
	progressEnums: string[];
};

const KEY = Symbol('attesta-row-context');

export function setRowContext(ctx: RowContext): void {
	setContext(KEY, ctx);
}

export function getRowContext(): RowContext {
	if (!hasContext(KEY)) {
		throw new Error('row context missing — mount TableFormShell first');
	}
	return getContext(KEY);
}
