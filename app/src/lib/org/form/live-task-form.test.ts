import { describe, expect, test } from 'vitest';
import type { Schema, UiSchemaRoot } from '@sjsf/form';

import { LiveTaskForm } from './live-task-form.svelte.js';
import type { TaskFormPlan } from './task-form-plan.js';

function planOf(partial: {
	projectionKey: string;
	schema?: Schema;
	derived?: TaskFormPlan['derived'];
}): TaskFormPlan {
	const schema = partial.schema ?? ({ type: 'object', properties: {} } as Schema);
	const uiSchema = {} as UiSchemaRoot;
	return {
		surface: 'live',
		schemas: { schema, uiSchema },
		...(partial.derived !== undefined ? { derived: partial.derived } : {}),
		chrome: [],
		chromeFieldIds: [],
		projectionKey: partial.projectionKey,
		sealSubmission: (value) => value
	};
}

const ports = {
	getValue: () => ({}),
	setValue: () => {},
	getDomain: () => ({})
};

describe('LiveTaskForm.updatePlan', () => {
	test('swaps plan in place without disposing chrome (deferForm)', async () => {
		const live = new LiveTaskForm(planOf({ projectionKey: 'a' }), ports, {
			deferForm: true
		});
		expect(live.plan.projectionKey).toBe('a');
		expect(live.form).toBeUndefined();

		live.updatePlan(
			planOf({
				projectionKey: 'b',
				schema: { type: 'object', properties: { x: { type: 'string' } } } as Schema
			})
		);
		expect(live.plan.projectionKey).toBe('b');
		expect(live.plan.schemas.schema).toMatchObject({
			properties: { x: { type: 'string' } }
		});

		await live.whenSettled();
		live.dispose();
	});

	test('same projectionKey is a no-op (avoids effect↔draft sync loop)', () => {
		const first = planOf({
			projectionKey: 'same',
			schema: { type: 'object', properties: { a: { type: 'string' } } } as Schema
		});
		const live = new LiveTaskForm(first, ports, { deferForm: true });
		live.updatePlan(
			planOf({
				projectionKey: 'same',
				schema: { type: 'object', properties: { b: { type: 'number' } } } as Schema
			})
		);
		expect(live.plan).toBe(first);
		live.dispose();
	});

	test('replaceDerived path keeps same LiveTaskForm instance', () => {
		const live = new LiveTaskForm(
			planOf({
				projectionKey: 'd1',
				derived: { 'a.b': { expr: '1' } }
			}),
			ports,
			{ deferForm: true }
		);
		const before = live;
		live.updatePlan(
			planOf({
				projectionKey: 'd2',
				derived: { 'a.b': { expr: '2' }, c: { expr: '"x"' } }
			})
		);
		expect(live).toBe(before);
		expect(live.plan.projectionKey).toBe('d2');
		expect(live.plan.derived).toEqual({
			'a.b': { expr: '2' },
			c: { expr: '"x"' }
		});
		live.dispose();
	});
});

describe('createConfigForm reactive schema wiring', () => {
	test('passes schema/uiSchema getters to createForm (source probe)', async () => {
		const { readFileSync } = await import('node:fs');
		const { dirname, join } = await import('node:path');
		const { fileURLToPath } = await import('node:url');
		const src = readFileSync(
			join(dirname(fileURLToPath(import.meta.url)), './create-config-form.svelte.ts'),
			'utf8'
		);
		expect(src).toMatch(/get schema\(\)/);
		expect(src).toMatch(/get uiSchema\(\)/);
		expect(src).toContain('ConfigFormSchemas | (() => ConfigFormSchemas)');
	});
});
