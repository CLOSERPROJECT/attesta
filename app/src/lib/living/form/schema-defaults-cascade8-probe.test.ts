/**
 * Cascade #8: form open must surface authored date sentinel (`$today` → today).
 *
 * Root: createConfigForm always passes sjsf `value` bind; createForm then
 * skips mergeFormDataAndSchemaDefaults. Empty Draft → empty widget value.
 * Fix: resolve `$today` once, pass resolved schema into merge + createForm
 * so sjsf re-merge/reset cannot inject the sentinel literal.
 *
 * Authored schema comes from feragame-config YAML (Go ships the same shape in the bundle).
 * Plant path: truckArrive carries `$today` on date (lot register no longer owns that field).
 */
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import feragameConfigYaml from '$living-fixtures/feragame-config.yaml?raw';
import { createAjvFormValidator } from './ajv-form-validator.js';
import { loadFeragameConfig, taskConfig } from './load-feragame-config.js';
import {
	localCalendarDateISO,
	mergeSchemaDefaultsIntoFormData,
	resolveSchemaDateDefaultSentinels,
	TODAY_DEFAULT_SENTINEL
} from './schema-defaults.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('cascade #8 schema defaults on truckArrive open', () => {
	const config = loadFeragameConfig(feragameConfigYaml);
	const authored = taskConfig(config.jobs.intake, 'truckArrive')!;
	const schema = authored.schema as Schema;
	const props = (schema.properties ?? {}) as Record<string, Schema>;

	test('authored date $today survives bundle schema', () => {
		expect(props.date?.default).toBe(TODAY_DEFAULT_SENTINEL);
		expect(props.date?.format).toBe('date');
	});

	test('empty-draft truckArrive open value includes today’s ISO date', () => {
		const today = localCalendarDateISO();
		const merged = mergeSchemaDefaultsIntoFormData(schema, {}, today);
		expect(merged).toMatchObject({
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
			formData: { turno: '1st' },
			initialDefaultsGenerated: true
		}) as Record<string, unknown>;
		expect(remerged.date).toBe(today);
		expect(remerged.date).not.toBe(TODAY_DEFAULT_SENTINEL);
	});

	test('partial draft date is not clobbered by schema defaults', () => {
		const today = localCalendarDateISO();
		const merged = mergeSchemaDefaultsIntoFormData(
			schema,
			{ date: '2020-01-15', turno: '2nd' },
			today
		);
		expect(merged).toMatchObject({
			date: '2020-01-15',
			turno: '2nd'
		});
	});
});
