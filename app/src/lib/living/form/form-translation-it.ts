import { fromRecord } from '@sjsf/form/lib/resolver';

/** Hardcoded Italian sjsf chrome for Attesta — not a locale switcher. */
const definitions = {
	submit: 'Invia',
	'array-schema-missing-items': 'Definizione items mancante',
	yes: 'Sì',
	no: 'No',
	'multi-schema-option-label-with-title': ({ title, index }: { title: string; index: number }) =>
		`${title} opzione ${index + 1}`,
	'multi-schema-option-label': ({ index }: { index: number }) => `Opzione ${index + 1}`,
	'add-array-item': 'Aggiungi elemento',
	'copy-array-item': 'Copia',
	'add-object-property': 'Aggiungi proprietà',
	'move-array-item-down': 'Giù',
	'move-array-item-up': 'Su',
	'remove-array-item': 'Elimina',
	'remove-object-property': 'Elimina',
	edit: 'Modifica',
	clear: 'Svuota',
	'validation-process-error': ({ error }: { error: { reason: string } }) =>
		FAILURE_REASONS[error.reason] ?? FAILURE_REASONS.error,
	'component-not-found': ({ type }: { type: string }) => `Componente "${type}" non trovato`,
	'key-input-title': ({ name }: { name: string }) => `Chiave ${name}`,
	'additional-property': 'Proprietà aggiuntiva',
	'unknown-field-error': ({ schema }: { schema: unknown }) =>
		`Schema JSON insufficiente per determinare il tipo.\n\nJSON Schema:\n${JSON.stringify(schema, null, 2)}`,
	expand: 'Espandi'
};

const FAILURE_REASONS: Record<string, string> = {
	aborted: 'Validazione interrotta',
	timeout: 'Validazione terminata per timeout',
	error: 'Errore durante la validazione'
};

export const translation = fromRecord(definitions);
