import {
	createFormValidator,
	DEFAULT_AJV_CONFIG
} from '@sjsf/ajv8-validator';
import type { FormValidator, ValidatorFactoryOptions } from '@sjsf/form';
import { normalizeFormCellMaps } from './form-cell-map.js';

/**
 * Draft validator: real Ajv (not noop) so JSON Schema `if`/`then` and
 * related keywords apply. `$data` is on for cheap cross-field rules later;
 * kernel seal Ajv lockstep stays out of this Attesta cut.
 *
 * createForm validates before onSubmit/seal — omit untouched cell-map cells
 * (`""` / undefined) so one filled sibling does not block submit.
 */
export function createAjvFormValidator<T>(
	_options?: ValidatorFactoryOptions
): FormValidator<T> {
	const inner = createFormValidator<T>({
		ajvOptions: { ...DEFAULT_AJV_CONFIG, $data: true }
	});
	return {
		...inner,
		isValid(schemaDef, rootSchema, formValue) {
			return inner.isValid(schemaDef, rootSchema, normalizeValidateValue(formValue));
		},
		validateFormValue(rootSchema, formValue) {
			return inner.validateFormValue(rootSchema, normalizeValidateValue(formValue));
		}
	};
}

function normalizeValidateValue<T>(formValue: T): T {
	if (formValue === null || typeof formValue !== 'object' || Array.isArray(formValue)) {
		return formValue;
	}
	return normalizeFormCellMaps(formValue as Record<string, unknown>) as T;
}
