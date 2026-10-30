import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Schema } from '@sjsf/form';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, substepConfig } from './load-truck-loop.js';
import { createAjvFormValidator } from './ajv-form-validator.js';

/** Gated field: when mode is `dual`, `detail` is required (JSON Schema if/then). */
const ifThenSchema = {
	type: 'object',
	properties: {
		mode: { type: 'string', enum: ['single', 'dual'] },
		detail: { type: 'string' }
	},
	if: {
		properties: { mode: { const: 'dual' } },
		required: ['mode']
	},
	then: { required: ['detail'] }
} as Schema;

describe('Ajv Draft validator', () => {
	test('createConfigForm path imports @sjsf/ajv8-validator, not noop', () => {
		const src = readFileSync(join(import.meta.dirname, 'create-config-form.svelte.ts'), 'utf8');
		expect(src).toContain("from './ajv-form-validator.js'");
		expect(src).not.toContain('@sjsf/form/validators/noop');
		const factory = readFileSync(join(import.meta.dirname, 'ajv-form-validator.ts'), 'utf8');
		expect(factory).toContain('@sjsf/ajv8-validator');
		expect(factory).toContain('$data: true');
	});

	test('if/then requires gated field only when condition holds', () => {
		const validator = createAjvFormValidator<Record<string, unknown>>();

		expect(validator.isValid(ifThenSchema, ifThenSchema, { mode: 'single' })).toBe(true);
		expect(
			validator.isValid(ifThenSchema, ifThenSchema, { mode: 'dual', detail: 'ok' })
		).toBe(true);

		expect(validator.isValid(ifThenSchema, ifThenSchema, { mode: 'dual' })).toBe(false);
	});
});

describe('containerReg Ajv dual-load', () => {
	const containerReg = substepConfig(
		loadTruckLoop(truckLoopStepsYaml).jobs.line,
		'containerReg'
	)!;
	const schema = containerReg.schema as Schema;
	const validator = createAjvFormValidator<Record<string, unknown>>();

	test('dual off on a row: sourceLoadId + weight is valid; form secondary optional', () => {
		expect(
			validator.isValid(schema, schema, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10, dualSources: false }]
			})
		).toBe(true);
		expect(
			validator.isValid(schema, schema, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10 }]
			})
		).toBe(true);
	});

	test('dual on a row: requires form secondary ≠ primary; no secondaryWeight', () => {
		expect(
			validator.isValid(schema, schema, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10, dualSources: true }]
			})
		).toBe(false);

		expect(
			validator.isValid(schema, schema, {
				sourceLoadId: 'load-a',
				secondarySourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10, dualSources: true }]
			})
		).toBe(false);

		expect(
			validator.isValid(schema, schema, {
				sourceLoadId: 'load-a',
				secondarySourceLoadId: 'load-b',
				polvereDiVetro: [{ weight: 10, dualSources: true }]
			})
		).toBe(true);
	});
});
