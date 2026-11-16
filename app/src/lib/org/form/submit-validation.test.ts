/**
 * Silent Invia: Ajv nested progress/<id> errors must become a visible path.
 */
import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Schema } from '@sjsf/form';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	formatSubmitValidationError,
	messagesAtExactPath,
	messagesUnderPath
} from './submit-validation.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('submit validation visibility', () => {
	/** Concrete properties (Go projection); string-only cells reject null. */
	const stringOnlyProgress = {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				properties: {
					a: { type: 'string', enum: ['partial', 'done'] },
					b: { type: 'string', enum: ['partial', 'done'] }
				},
				additionalProperties: false
			}
		}
	} as Schema;

	test('nested progress/<id> Ajv errors are invisible with exact-path only', () => {
		const validator = createAjvFormValidator<Record<string, unknown>>();
		const result = validator.validateFormValue(stringOnlyProgress, {
			progress: { a: 'partial', b: null }
		});
		const errors = result.errors ?? [];
		expect(errors.length).toBeGreaterThan(0);
		expect(messagesAtExactPath(errors, ['progress'])).toEqual([]);
		expect(messagesUnderPath(errors, ['progress']).length).toBeGreaterThan(0);
		expect(
			formatSubmitValidationError({ errors }),
			'onSubmitError banner must name nested path'
		).toMatch(/progress\.b/);
	});

	test('createConfigForm / mapTable / TaskForm wire onSubmitError path', () => {
		const createCfg = readFileSync(join(libDir, 'create-config-form.svelte.ts'), 'utf8');
		const mapSrc = readFileSync(join(libDir, 'map-table-field.svelte'), 'utf8');
		const task = readFileSync(join(libDir, '../TaskForm.svelte'), 'utf8');
		const ajvSrc = readFileSync(join(libDir, 'ajv-form-validator.ts'), 'utf8');
		const bundleSrc = readFileSync(join(libDir, 'task-form-bundle.ts'), 'utf8');
		expect(createCfg).toContain('onSubmitError');
		expect(mapSrc).toContain('getFieldComponent');
		expect(mapSrc).toContain('data-testid="map-table"');
		expect(mapSrc).not.toContain('rowTable');
		expect(mapSrc).not.toContain('progressEnums');
		expect(mapSrc).not.toContain('TableFormShell');
		expect(() => readFileSync(join(libDir, 'form-cell-map.ts'), 'utf8')).toThrow();
		expect(ajvSrc).not.toContain('normalizeFormCellMaps');
		expect(ajvSrc).not.toContain('form-cell-map');
		expect(bundleSrc).not.toContain('normalizeFormCellMaps');
		expect(bundleSrc).not.toContain('form-cell-map');
		expect(task).toContain('task-submit-error');
		expect(task).toContain('onSubmitError');
		expect(task).toContain('initialData');
		expect(task).toContain('planFromTaskFormBundle');
		expect(task).toContain('ConfigForm');
		expect(task).not.toContain('TableFormShell');
		expect(task).not.toContain('progressEnums');
		expect(task).not.toContain('compileTaskFormPlan');
		expect(task).not.toContain('seedMapTableDraft');
		expect(task).not.toContain('seedProgressDraftFromDomain');
	});
});
