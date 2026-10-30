import {
	createFormValidator,
	DEFAULT_AJV_CONFIG
} from '@sjsf/ajv8-validator';
import type { FormValidator, ValidatorFactoryOptions } from '@sjsf/form';

/**
 * Draft validator: real Ajv (not noop) so JSON Schema `if`/`then` and
 * related keywords apply. `$data` is on for cheap cross-field rules later;
 * kernel seal Ajv lockstep stays out of this Attesta cut.
 */
export function createAjvFormValidator<T>(
	_options?: ValidatorFactoryOptions
): FormValidator<T> {
	return createFormValidator<T>({
		ajvOptions: { ...DEFAULT_AJV_CONFIG, $data: true }
	});
}
