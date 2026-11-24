import { describe, expect, test } from 'vitest';
import type { UiSchemaRoot } from '@sjsf/form';

import { chromeForSeededFields, lockSeededUiSchema } from './lock-seeded-ui-schema.js';

describe('lockSeededUiSchema', () => {
	test('overrides enumField + stringField → readonlyField for seeded keys only', () => {
		const ui = {
			lotId: {
				'ui:components': { stringField: 'enumField' }
			},
			note: {
				'ui:components': { stringField: 'enumField' }
			}
		} as UiSchemaRoot;
		const locked = lockSeededUiSchema(ui, ['lotId']);
		expect(locked.lotId).toEqual({
			'ui:components': {
				stringField: 'readonlyField',
				enumField: 'readonlyField'
			},
			'ui:options': { chromeId: 'lotId' }
		});
		expect(locked.note).toEqual({
			'ui:components': { stringField: 'enumField' }
		});
	});

	test('empty seededFields returns same uiSchema reference', () => {
		const ui = { lotId: { 'ui:components': { stringField: 'enumField' } } } as UiSchemaRoot;
		expect(lockSeededUiSchema(ui, [])).toBe(ui);
	});

	test('can force hiddenField instead of readonly', () => {
		const ui = {
			lotId: { 'ui:components': { stringField: 'enumField' } }
		} as UiSchemaRoot;
		expect(lockSeededUiSchema(ui, ['lotId'], 'hiddenField').lotId).toEqual({
			'ui:components': {
				stringField: 'hiddenField',
				enumField: 'hiddenField'
			},
			'ui:options': {}
		});
	});
});

describe('chromeForSeededFields', () => {
	test('adds readonly chrome expr for seeded fields not already present', () => {
		expect(chromeForSeededFields(['lotId'], [{ id: 'hint', kind: 'readonly', expr: '1' }])).toEqual([
			{ id: 'hint', kind: 'readonly', expr: '1' },
			{ id: 'lotId', kind: 'readonly', expr: '$.lotId' }
		]);
	});

	test('skips ids already in chrome', () => {
		expect(
			chromeForSeededFields(['lotId'], [{ id: 'lotId', kind: 'readonly', expr: '$.other' }])
		).toEqual([{ id: 'lotId', kind: 'readonly', expr: '$.other' }]);
	});
});
