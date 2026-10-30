/**
 * Compact array item chrome for containerReg material rows:
 * orderable off, hide item/leaf titles, material-aware weight placeholder,
 * per-row dual checkbox-only + form-level primary/secondary load selects,
 * theme `compactArrayItemField` (delete-only toolbar).
 */

import { paintAuthoredFieldMarkers } from './authored-field-registry.js';
import {
	CONTAINER_REG_ITEM_FIELD_ORDER,
	CONTAINER_REG_MATERIALS
} from './container-reg-materials.js';

export const COMPACT_ARRAY_ITEM_FIELD = 'compactArrayItemField' as const;

const SHADCN_PLACEHOLDER_KEYS = ['shadcn4Text', 'shadcn4Number', 'shadcn4Select'] as const;

function asRecord(value: unknown): Record<string, unknown> | null {
	if (value === null || typeof value !== 'object' || Array.isArray(value)) return null;
	return value as Record<string, unknown>;
}

export function materialWeightPlaceholder(materialLabel: string): string {
	return `Peso ${materialLabel} (kg)…`;
}

export function sourceLoadIdPlaceholder(): string {
	return 'Seleziona carico…';
}

export function secondarySourceLoadIdPlaceholder(): string {
	return 'Seleziona carico secondario…';
}

function fieldHasPlaceholder(fieldUi: Record<string, unknown>): boolean {
	if (typeof fieldUi['ui:placeholder'] === 'string') return true;
	const opts = asRecord(fieldUi['ui:options']);
	if (!opts) return false;
	if (typeof opts.placeholder === 'string') return true;
	for (const key of SHADCN_PLACEHOLDER_KEYS) {
		const widget = asRecord(opts[key]);
		if (widget && typeof widget.placeholder === 'string') return true;
	}
	return false;
}

function withShadcnPlaceholder(
	existing: Record<string, unknown>,
	text: string
): Record<string, unknown> {
	if (fieldHasPlaceholder(existing)) {
		return existing;
	}
	const existingOpts = asRecord(existing['ui:options']) ?? {};
	const nextOpts: Record<string, unknown> = { ...existingOpts };
	for (const widgetKey of SHADCN_PLACEHOLDER_KEYS) {
		const prev = asRecord(existingOpts[widgetKey]) ?? {};
		nextOpts[widgetKey] = { ...prev, placeholder: text };
	}
	return { ...existing, 'ui:options': nextOpts };
}

function mergeHideTitle(existing: Record<string, unknown>): Record<string, unknown> {
	const existingOpts = asRecord(existing['ui:options']) ?? {};
	return {
		...existing,
		'ui:options': {
			...existingOpts,
			hideTitle: true
		}
	};
}

/** Belt for compat resolver: string + enum → enumField / selectWidget. */
function withEnumFieldComponent(existing: Record<string, unknown>): Record<string, unknown> {
	const components = asRecord(existing['ui:components']) ?? {};
	return {
		...existing,
		'ui:components': {
			...components,
			stringField: 'enumField'
		}
	};
}

function paintMaterialArray(
	uiSchema: Record<string, unknown>,
	key: string,
	label: string
): void {
	const existing = asRecord(uiSchema[key]) ?? {};
	const existingComponents = asRecord(existing['ui:components']) ?? {};
	const existingOpts = asRecord(existing['ui:options']) ?? {};
	const existingLayouts = asRecord(existingOpts.layouts) ?? {};
	const existingArrayFieldLayout = asRecord(existingLayouts['array-field']) ?? {};
	const existingArrayItemsLayout = asRecord(existingLayouts['array-items']) ?? {};
	const existingTitleRowLayout = asRecord(existingLayouts['array-field-title-row']) ?? {};
	const existingItems = asRecord(existing.items) ?? {};
	const existingItemOpts = asRecord(existingItems['ui:options']) ?? {};
	const existingItemLayouts = asRecord(existingItemOpts.layouts) ?? {};
	const existingObjectPropsLayout = asRecord(existingItemLayouts['object-properties']) ?? {};
	const existingWeight = asRecord(existingItems.weight) ?? {};
	const existingDual = asRecord(existingItems.dualSources) ?? {};

	const weightWithTitle = mergeHideTitle(existingWeight);
	const weightPainted = withShadcnPlaceholder(
		weightWithTitle,
		materialWeightPlaceholder(label)
	);
	const dualPainted = paintAuthoredFieldMarkers({
		...existingDual,
		'ui:field': 'checkbox-only'
	});

	uiSchema[key] = {
		...existing,
		'ui:components': {
			...existingComponents,
			arrayItemField: COMPACT_ARRAY_ITEM_FIELD
		},
		'ui:options': {
			...existingOpts,
			orderable: false,
			copyable: false,
			layouts: {
				...existingLayouts,
				'array-field': {
					...existingArrayFieldLayout,
					class:
						'gap-0 has-[[data-layout=array-items]>*]:gap-2 [&>button]:w-auto [&>button]:self-start has-[[data-layout=array-items]>*]:[&_[data-layout=array-field-title-row]]:mb-3'
				},
				'array-items': {
					...existingArrayItemsLayout,
					class: 'gap-1'
				},
				'array-field-title-row': {
					...existingTitleRowLayout,
					variant: 'label',
					class: 'mb-1'
				}
			}
		},
		items: {
			...mergeHideTitle(existingItems),
			'ui:options': {
				...asRecord(mergeHideTitle(existingItems)['ui:options']),
				order: [...CONTAINER_REG_ITEM_FIELD_ORDER],
				layouts: {
					...existingItemLayouts,
					'object-properties': {
						...existingObjectPropsLayout,
						/* Weight flexes; dual checkbox stays fixed beside it. */
						class: 'flex w-full flex-row flex-nowrap items-stretch gap-2'
					}
				}
			},
			weight: weightPainted,
			dualSources: dualPainted
		}
	};
}

/** Mutates uiSchema: form-level load selects + material arrays with dual checkbox-only. */
export function paintContainerRegCompactArrayUi(uiSchema: Record<string, unknown>): void {
	const sourceExisting = asRecord(uiSchema.sourceLoadId) ?? {};
	uiSchema.sourceLoadId = withEnumFieldComponent(
		withShadcnPlaceholder(sourceExisting, sourceLoadIdPlaceholder())
	);

	const secondaryExisting = asRecord(uiSchema.secondarySourceLoadId) ?? {};
	uiSchema.secondarySourceLoadId = withEnumFieldComponent(
		withShadcnPlaceholder(secondaryExisting, secondarySourceLoadIdPlaceholder())
	);

	for (const { key, label } of CONTAINER_REG_MATERIALS) {
		paintMaterialArray(uiSchema, key, label);
	}
}
