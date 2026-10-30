/** Fixed plant materials for containerReg — schema keys + human labels as entity facts. */

export type ContainerRegMaterial = {
	readonly key: string;
	readonly label: string;
};

export const CONTAINER_REG_MATERIALS: readonly ContainerRegMaterial[] = [
	{ key: 'polvereDiVetro', label: 'Polvere di vetro' },
	{ key: 'granellaDiVetro', label: 'Granella di vetro' },
	{ key: 'granellaDiVetro2', label: 'Granella di vetro 2' },
	{ key: 'polvereDiSilicio', label: 'Polvere di silicio' },
	{ key: 'plastica', label: 'Plastica' },
	{ key: 'alluminio', label: 'Alluminio' },
	{ key: 'sopravaglio', label: 'Sopravaglio' },
	{ key: 'polvereDiAspirazione', label: 'Polvere di aspirazione' }
] as const;

export const CONTAINER_REG_MATERIAL_KEYS = CONTAINER_REG_MATERIALS.map((m) => m.key);

/** uiSchema order: primary + form-level secondary load, then fixed material sections. */
export const CONTAINER_REG_FIELD_ORDER = [
	'sourceLoadId',
	'secondarySourceLoadId',
	...CONTAINER_REG_MATERIAL_KEYS
] as const;

/** Compact row field order inside each material array item (dual checkbox only). */
export const CONTAINER_REG_ITEM_FIELD_ORDER = ['weight', 'dualSources'] as const;
