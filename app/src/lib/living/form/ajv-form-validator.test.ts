import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Schema } from '@sjsf/form';
import feragameConfigYaml from '$living-fixtures/feragame-config.yaml?raw';
import { loadFeragameConfig, substepConfig } from './load-feragame-config.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import type { TaskConfig } from '../types.js';

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

describe('containerReg register Ajv', () => {
	const group = substepConfig(loadFeragameConfig(feragameConfigYaml).jobs.line, 'containerReg')!;
	// Fixture groups still author nested leaves as `substeps` in YAML (SPA parse keeps that key).
	const register = (
		(group.substeps as TaskConfig[] | undefined) ??
		(group.tasks as TaskConfig[] | undefined) ??
		[]
	).find((t) => t.id === 'register');
	const schema = (register?.schema ?? null) as Schema | null;
	const uiSchema = (register?.uiSchema ?? null) as Record<string, unknown> | null;
	const validator = createAjvFormValidator<Record<string, unknown>>();

	test('sourceLoadId + weight valid; dualSources / secondary enum gone', () => {
		expect(schema).toBeTruthy();
		expect(schema).not.toHaveProperty('if');
		const props = (schema as { properties?: Record<string, unknown> }).properties ?? {};
		expect(props).not.toHaveProperty('secondarySourceLoadId');
		expect(props).not.toHaveProperty('dualSources');
		const rowProps =
			(
				(props.polvereDiVetro as { items?: { properties?: Record<string, unknown> } } | undefined)
					?.items?.properties
			) ?? {};
		expect(rowProps).not.toHaveProperty('dualSources');
		expect(rowProps).toHaveProperty('continuesContainerId');

		expect(
			validator.isValid(schema!, schema!, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10, incomplete: false }]
			})
		).toBe(true);
		expect(
			validator.isValid(schema!, schema!, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ weight: 10, continuesContainerId: 'C-1', incomplete: false }]
			})
		).toBe(true);
		expect(
			validator.isValid(schema!, schema!, {
				sourceLoadId: 'load-a',
				polvereDiVetro: [{ incomplete: false }]
			})
		).toBe(false);
	});

	test('continuesContainerId authored as hiddenField (not RJSF ui:widget)', () => {
		expect(uiSchema).toBeTruthy();
		const contUi =
			(
				uiSchema!.polvereDiVetro as
					| {
							items?: {
								continuesContainerId?: {
									'ui:components'?: { stringField?: string };
									'ui:widget'?: unknown;
								};
							};
					  }
					| undefined
			)?.items?.continuesContainerId ?? {};
		expect(contUi['ui:components']?.stringField).toBe('hiddenField');
		expect(contUi).not.toHaveProperty('ui:widget');
		const themeSrc = readFileSync(join(import.meta.dirname, 'form-theme.ts'), 'utf8');
		expect(themeSrc).toContain('hiddenField: HiddenField');
	});
});
