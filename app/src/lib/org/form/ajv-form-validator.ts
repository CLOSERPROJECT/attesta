import {
	createFormValidator,
	DEFAULT_AJV_CONFIG
} from '@sjsf/ajv8-validator';
import type {
	FieldValueValidator,
	FormValueValidator,
	Validator,
	ValidatorFactoryOptions
} from '@sjsf/form';
import localizeEn from 'ajv-i18n/localize/en';
import localizeIt from 'ajv-i18n/localize/it';
import { getLocale, type Locale } from '#lib/paraglide/runtime.js';

/**
 * Draft validator: real Ajv (not noop) so JSON Schema `if`/`then` and
 * related keywords apply. `$data` is on for cheap cross-field rules later;
 * kernel seal Ajv lockstep stays out of this Attesta cut.
 *
 * `messages: false` + ajv-i18n so field/submit errors follow Paraglide locale
 * (Italian default; English when engineering cookie is `en`).
 *
 * No SPA cell-map strip — Draft validates as-is; Go prep owns Commit omit.
 */
export function createAjvFormValidator<T>(
	_options?: ValidatorFactoryOptions,
	locale: Locale = getLocale()
): Validator & FormValueValidator<T> & FieldValueValidator {
	const localize = locale === 'en' ? localizeEn : localizeIt;
	return createFormValidator<T>({
		ajvOptions: { ...DEFAULT_AJV_CONFIG, $data: true, messages: false },
		localize: (errors) => {
			localize(errors);
			return errors;
		}
	});
}
