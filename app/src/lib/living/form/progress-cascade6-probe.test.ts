/**
 * Cascade #6 regression: one row enum + untouched siblings.
 * No SPA normalize — Ajv sees Draft as-is. Happy path = omit /
 * undefined. Empty string must fail cell enum. Stock clearable drops
 * the draft key; enum options are real values only.
 *
 * Projected schema mirrors Go concrete properties (not propertyNames +
 * additionalProperties template). sealSubmission is identity.
 */
import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { Schema, UiSchemaRoot } from '@sjsf/form';
import { orderProperties } from '@sjsf/form/core';
import { createAjvFormValidator } from './ajv-form-validator.js';
import { planFromTaskFormBundle, type TaskFormBundle } from './task-form-bundle.js';

const libDir = dirname(fileURLToPath(import.meta.url));

const cellSchema = {
	enum: ['partial', 'done']
};

/** Fixture: lineLoad mapTable after Go concrete-property projection. */
function lineLoadProjectedSchema(rowIds: string[]): Schema {
	const properties: Record<string, typeof cellSchema> = {};
	for (const id of rowIds) properties[id] = cellSchema;
	return {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				properties,
				additionalProperties: false
			}
		}
	} as Schema;
}

function sealPlan() {
	const rowIds = ['NIR-1.1', 'NIR-1.2', 'NIR-1.3'];
	const bundle: TaskFormBundle = {
		jobId: 'line',
		taskId: 'lineLoad',
		jobInstanceId: '1st',
		title: 'lineLoad',
		surface: 'live',
		schema: lineLoadProjectedSchema(rowIds),
		uiSchema: {} as UiSchemaRoot,
		initialData: {},
		rowIds,
		chrome: [],
		chromeFieldIds: [],
		sealFieldIds: [],
		bindings: {}
	};
	return planFromTaskFormBundle(bundle);
}

describe('cascade #6 no SPA cell-map normalize', () => {
	const rowIds = ['NIR-1.1', 'NIR-1.2'];
	const schema = lineLoadProjectedSchema(rowIds);
	const validator = createAjvFormValidator<Record<string, unknown>>();

	test('form-cell-map normalize helpers are gone', () => {
		expect(() => readFileSync(join(libDir, 'form-cell-map.ts'), 'utf8')).toThrow();
		const ajvSrc = readFileSync(join(libDir, 'ajv-form-validator.ts'), 'utf8');
		const bundleSrc = readFileSync(join(libDir, 'task-form-bundle.ts'), 'utf8');
		expect(ajvSrc).not.toContain('normalizeFormCellMaps');
		expect(ajvSrc).not.toContain('form-cell-map');
		expect(bundleSrc).not.toContain('normalizeFormCellMaps');
		expect(bundleSrc).not.toContain('form-cell-map');
	});

	test('createAjvFormValidator: sibling "" fails cell enum (no strip)', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'done', [sibling]: '' };
		expect(validator.isValid(schema, schema, { progress })).toBe(false);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors?.length).toBeGreaterThan(0);
	});

	test('createAjvFormValidator: omit siblings validates', () => {
		const [touched] = rowIds;
		const progress = { [touched]: 'done' };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
	});

	test('createAjvFormValidator: enum + own-property undefined validates', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'done', [sibling]: undefined };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
	});

	test('null sibling fails (not Domain clear)', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'partial', [sibling]: null };
		expect(validator.isValid(schema, schema, { progress })).toBe(false);
	});

	test('sealSubmission is identity (no cell-map strip)', () => {
		const plan = sealPlan();
		const withEmpty = {
			progress: { 'NIR-1.1': 'done', 'NIR-1.2': '', 'NIR-1.3': undefined }
		};
		expect(plan.sealSubmission(withEmpty)).toEqual(withEmpty);
	});

	test('mapTable row keys === projected ids when Draft omits the map', () => {
		const projectedIds = ['NIR-1.1', 'NIR-1.2', 'NIR-1.3'];
		const progressSchema = (
			lineLoadProjectedSchema(projectedIds) as {
				properties?: { progress?: { properties?: Record<string, unknown> } };
			}
		).properties?.progress;
		const concreteProperties = progressSchema?.properties ?? {};
		const draftOmitsMap: Record<string, unknown> = {};

		expect(Object.keys(draftOmitsMap)).toEqual([]);
		expect(orderProperties(concreteProperties, undefined)).toEqual(projectedIds);

		const mapSrc = readFileSync(join(libDir, 'map-table-field.svelte'), 'utf8');
		expect(mapSrc).toContain('objCtx.propertiesOrder()');
		expect(mapSrc).toMatch(/\{#each objCtx\.propertiesOrder\(\) as rowKey/);
	});

	test('mapTable scalar bind: omit quiet + stock clear drops key; no null via setValue', () => {
		const mapSrc = readFileSync(join(libDir, 'map-table-field.svelte'), 'utf8');
		expect(mapSrc).toContain('MapTableCells');
		expect(mapSrc).toContain('cells.setScalarCell');
		expect(mapSrc).toContain('cells.setCell');
		expect(mapSrc).not.toContain('$effect');

		const cellsSrc = readFileSync(join(libDir, 'map-table-cells.svelte.ts'), 'utf8');
		expect(cellsSrc).toContain('current === undefined && cell === undefined');
		expect(cellsSrc).toContain('delete out[rowKey]');
		expect(cellsSrc).not.toContain('out[rowKey] = null');
		expect(cellsSrc).not.toContain('cell === undefined ? null');
		const scalarStart = cellsSrc.indexOf('setScalarCell');
		const objectStart = cellsSrc.indexOf('setCell');
		expect(scalarStart).toBeGreaterThan(-1);
		expect(objectStart).toBeGreaterThan(scalarStart);
		const scalarBody = cellsSrc.slice(scalarStart, objectStart);
		expect(scalarBody).not.toContain('isQuietEmptyCell');
		expect(cellsSrc).not.toContain('$effect');
	});
});
