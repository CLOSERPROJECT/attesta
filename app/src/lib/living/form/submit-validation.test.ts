/**
 * Silent Invia: Ajv nested progress/<id> errors must become a visible path.
 */
import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	formatSubmitValidationError,
	messagesAtExactPath,
	messagesUnderPath
} from './submit-validation.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('submit validation visibility', () => {
	const stringOnlyProgress = {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				propertyNames: { enum: ['a', 'b'] },
				additionalProperties: { type: 'string', enum: ['partial', 'done'] }
			}
		}
	};

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
		const cellMap = readFileSync(join(libDir, 'form-cell-map.ts'), 'utf8');
		expect(createCfg).toContain('onSubmitError');
		expect(mapSrc).toContain('getFieldComponent');
		expect(mapSrc).toContain('data-testid="map-table"');
		expect(cellMap).toContain('export function omitUntouchedCells');
		expect(cellMap).toContain('export function normalizeFormCellMaps');
		expect(cellMap).not.toContain('progressEnumsFromFormSchema');
		expect(cellMap).not.toContain('rowValueTypeFromSchema');
		expect(task).toContain('task-submit-error');
		expect(task).toContain('onSubmitError');
		expect(task).toContain('findTaskFormBundle');
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
