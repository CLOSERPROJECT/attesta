import { describe, expect, test } from 'vitest';
import { createTranslate } from '@sjsf/form';
import { getSjsfTranslation } from './sjsf-translation.js';

describe('getSjsfTranslation', () => {
	test('Italian labels for stock chrome', () => {
		const translate = createTranslate(getSjsfTranslation('it'));
		expect(translate('submit', {})).toBe('Invia');
		expect(translate('add-array-item', {})).toBe('Aggiungi elemento');
		expect(translate('remove-array-item', {})).toBe('Rimuovi');
		expect(translate('yes', {})).toBe('Sì');
		expect(translate('no', {})).toBe('No');
		expect(translate('multi-schema-option-label', { index: 0 })).toBe('Opzione 1');
		expect(
			translate('validation-process-error', {
				error: { reason: 'aborted' } as never
			})
		).toBe('Validazione interrotta');
	});

	test('English labels when locale is en', () => {
		const translate = createTranslate(getSjsfTranslation('en'));
		expect(translate('submit', {})).toBe('Submit');
		expect(translate('add-array-item', {})).toBe('Add item');
		expect(translate('yes', {})).toBe('Yes');
	});

	test('falls back to upstream en for unknown-field-error', () => {
		const translate = createTranslate(getSjsfTranslation('it'));
		const msg = translate('unknown-field-error', { schema: { type: 'string' } });
		expect(msg).toContain('JSON Schema');
	});
});
