/**
 * Cascade #8: register form open must surface JSON Schema `default`
 * (`idoneitaRiutilizzo: NIR`) and authored date sentinel (`$today` → today).
 *
 * Root: createConfigForm always passes sjsf `value` bind; createForm then
 * skips mergeFormDataAndSchemaDefaults. Empty Draft → empty widget value.
 * Fix: resolve `$today` once, pass resolved schema into merge + createForm
 * so sjsf re-merge/reset cannot inject the sentinel literal.
 *
 * Authored schema comes from truck-loop YAML (Go ships the same shape in the bundle).
 */
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { createAjvFormValidator } from './ajv-form-validator.js';
import { loadTruckLoop, taskConfig } from './load-truck-loop.js';
import {
	localCalendarDateISO,
	mergeSchemaDefaultsIntoFormData,
	resolveSchemaDateDefaultSentinels,
	TODAY_DEFAULT_SENTINEL
} from './schema-defaults.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('cascade #8 schema defaults on register open', () => {
	const config = loadTruckLoop(truckLoopStepsYaml);
	const authored = taskConfig(config.jobs.intake, 'register')!;
	const schema = authored.schema as Schema;
	const props = (schema.properties ?? {}) as Record<string, Schema>;

	test('authored idoneitaRiutilizzo default NIR and date $today survive bundle schema', () => {
		expect(props.idoneitaRiutilizzo?.default).toBe('NIR');
		expect(props.date?.default).toBe(TODAY_DEFAULT_SENTINEL);
		expect(props.date?.format).toBe('date');
	});

	test('empty-draft register open value includes NIR and today’s ISO date', () => {
		const today = localCalendarDateISO();
		const merged = mergeSchemaDefaultsIntoFormData(schema, {}, today);
		expect(merged).toMatchObject({
			idoneitaRiutilizzo: 'NIR',
			date: today
		});

		const createSrc = readFileSync(join(libDir, './create-config-form.svelte.ts'), 'utf8');
		expect(createSrc).toContain('resolveSchemaDateDefaultSentinels');
		expect(createSrc).toContain('mergeSchemaDefaultsIntoFormData');
		expect(createSrc).toMatch(/value:\s*\[/);
	});

	test('missing date re-merge on resolved schema yields ISO, never $today', () => {
		const today = localCalendarDateISO();
		const resolved = resolveSchemaDateDefaultSentinels(schema, today);
		const dateDefault = (resolved.properties as Record<string, Schema> | undefined)?.date
			?.default;
		expect(dateDefault).toBe(today);
		expect(dateDefault).not.toBe(TODAY_DEFAULT_SENTINEL);

		const validator = createAjvFormValidator();
		const merger = createFormMerger({ validator, schema: resolved });
		const remerged = merger.mergeFormDataAndSchemaDefaults({
			schema: resolved,
			formData: { idoneitaRiutilizzo: 'NIR' },
			initialDefaultsGenerated: true
		}) as Record<string, unknown>;
		expect(remerged.date).toBe(today);
		expect(remerged.date).not.toBe(TODAY_DEFAULT_SENTINEL);
	});

	test('partial draft IR and date are not clobbered by schema defaults', () => {
		const today = localCalendarDateISO();
		const merged = mergeSchemaDefaultsIntoFormData(
			schema,
			{ idoneitaRiutilizzo: 'IR', date: '2020-01-15' },
			today
		);
		expect(merged.idoneitaRiutilizzo).toBe('IR');
		expect(merged.date).toBe('2020-01-15');
	});

	test('format:date without $today does not inject a calendar default', () => {
		const bareDateSchema = {
			type: 'object',
			properties: {
				date: { type: 'string', format: 'date', title: 'Data' },
				idoneitaRiutilizzo: { type: 'string', default: 'NIR' }
			}
		} as Schema;
		const merged = mergeSchemaDefaultsIntoFormData(bareDateSchema, {}, '2099-12-31');
		expect(merged.idoneitaRiutilizzo).toBe('NIR');
		expect(merged.date).toBeUndefined();
	});
});
