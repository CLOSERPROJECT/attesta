import type { Translation } from '@sjsf/form';
import { chain, fromRecord } from '@sjsf/form/lib/resolver';
import { translation as en } from '@sjsf/form/translations/en';
import * as m from '#lib/paraglide/messages.js';
import type { Locale } from '#lib/paraglide/runtime.js';

/**
 * Paraglide-backed sjsf Labels table. Chains to upstream `en` so any missing
 * id (e.g. developer-facing `unknown-field-error`) still resolves.
 */
export function getSjsfTranslation(locale: Locale): Translation {
	const opt = { locale };
	return chain(
		fromRecord({
			submit: m.form_submit({}, opt),
			'add-array-item': m.form_add_array_item({}, opt),
			'remove-array-item': m.form_remove_array_item({}, opt),
			'copy-array-item': m.form_copy_array_item({}, opt),
			'move-array-item-up': m.form_move_array_item_up({}, opt),
			'move-array-item-down': m.form_move_array_item_down({}, opt),
			'add-object-property': m.form_add_object_property({}, opt),
			'remove-object-property': m.form_remove_object_property({}, opt),
			yes: m.form_yes({}, opt),
			no: m.form_no({}, opt),
			expand: m.form_expand({}, opt),
			edit: m.form_edit({}, opt),
			clear: m.form_clear({}, opt),
			'additional-property': m.form_additional_property({}, opt),
			'array-schema-missing-items': m.form_array_schema_missing_items({}, opt),
			'multi-schema-option-label': ({ index }: { index: number }) =>
				m.form_multi_schema_option_label({ index: index + 1 }, opt),
			'multi-schema-option-label-with-title': ({
				title,
				index
			}: {
				title: string;
				index: number;
			}) =>
				m.form_multi_schema_option_label_with_title(
					{ title, index: index + 1 },
					opt
				),
			'key-input-title': ({ name }: { name: string }) =>
				m.form_key_input_title({ name }, opt),
			'component-not-found': ({ type }: { type: string }) =>
				m.form_component_not_found({ type }, opt),
			'validation-process-error': ({
				error
			}: {
				error: { reason: string };
			}) => {
				switch (error.reason) {
					case 'aborted':
						return m.form_validation_aborted({}, opt);
					case 'timeout':
						return m.form_validation_timeout({}, opt);
					default:
						return m.form_validation_error({}, opt);
				}
			}
		}),
		en
	);
}
