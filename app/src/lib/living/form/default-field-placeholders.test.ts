import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import {
	defaultPlaceholderText,
	injectDefaultPlaceholders
} from './default-field-placeholders.js';

describe('defaultPlaceholderText', () => {
	test('uses schema title when present', () => {
		expect(defaultPlaceholderText('client', { type: 'string', title: 'Cliente' })).toBe(
			'Inserisci Cliente…'
		);
	});

	test('falls back to property key when title missing', () => {
		expect(defaultPlaceholderText('truckWeight', { type: 'number' })).toBe(
			'Inserisci truckWeight…'
		);
	});
});

describe('injectDefaultPlaceholders', () => {
	test('injects shadcn4 placeholders for string and number fields missing one', () => {
		const uiSchema: Record<string, unknown> = {
			'ui:options': { order: ['client', 'truckWeight'] }
		};
		const properties: Record<string, Schema> = {
			client: { type: 'string', title: 'Cliente' },
			truckWeight: { type: 'number', title: 'Peso totale camion' }
		};

		injectDefaultPlaceholders(uiSchema, properties, new Set());

		expect(uiSchema.client).toEqual({
			'ui:options': {
				shadcn4Text: { placeholder: 'Inserisci Cliente…' },
				shadcn4Number: { placeholder: 'Inserisci Cliente…' },
				shadcn4Select: { placeholder: 'Inserisci Cliente…' }
			}
		});
		expect(uiSchema.truckWeight).toEqual({
			'ui:options': {
				shadcn4Text: { placeholder: 'Inserisci Peso totale camion…' },
				shadcn4Number: { placeholder: 'Inserisci Peso totale camion…' },
				shadcn4Select: { placeholder: 'Inserisci Peso totale camion…' }
			}
		});
	});

	test('never overwrites authored placeholders', () => {
		const uiSchema: Record<string, unknown> = {
			client: {
				'ui:options': {
					placeholder: 'Type client name',
					shadcn4Text: { placeholder: 'Type client name' }
				}
			}
		};
		injectDefaultPlaceholders(
			uiSchema,
			{ client: { type: 'string', title: 'Cliente' } },
			new Set()
		);
		expect(uiSchema.client).toEqual({
			'ui:options': {
				placeholder: 'Type client name',
				shadcn4Text: { placeholder: 'Type client name' }
			}
		});
	});

	test('skips chrome ids, readOnly, and non-input types', () => {
		const uiSchema: Record<string, unknown> = {};
		const properties: Record<string, Schema> = {
			registeredData: { type: 'string', title: 'Dati registrati', readOnly: true },
			flag: { type: 'boolean', title: 'Flag' },
			weights: { type: 'object', title: 'Weights' },
			client: { type: 'string', title: 'Cliente' }
		};

		injectDefaultPlaceholders(uiSchema, properties, new Set(['registeredData']));

		expect(uiSchema.registeredData).toBeUndefined();
		expect(uiSchema.flag).toBeUndefined();
		expect(uiSchema.weights).toBeUndefined();
		expect(
			(uiSchema.client as { 'ui:options': { shadcn4Text: { placeholder: string } } })[
				'ui:options'
			].shadcn4Text.placeholder
		).toBe('Inserisci Cliente…');
	});
});
