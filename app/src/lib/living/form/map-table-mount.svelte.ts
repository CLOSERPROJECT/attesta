/**
 * mapTable Substep mount: projected leftovers + authored Substep → live FormState
 * (+ ConfigForm). Owns expand Derived, row seed, `$domain` (tareKinds), and
 * createConfigForm. Session keeps Draft + Submission; mount bugs stay here.
 */

import {
	createConfigForm,
	type CreateConfigFormResult
} from './create-config-form.svelte.js';
import {
	prepareMapTableForm,
	type MapTableAuthoredSubstep,
	type MapTableProjectedForm
} from './map-table-mount.js';

export {
	seedMapTableDraft,
	prepareMapTableForm,
	type MapTableAuthoredSubstep,
	type MapTableProjectedForm
} from './map-table-mount.js';

/** Live FormState (+ ConfigForm) for a projected mapTable Substep. */
export function mountMapTableForm<T extends Record<string, unknown>>(args: {
	projected: MapTableProjectedForm;
	substep: MapTableAuthoredSubstep;
	getValue: () => T;
	setValue: (next: T) => void;
	onSubmit?: (value: T) => void | Promise<void>;
}): CreateConfigFormResult<T> {
	const { schemas, options } = prepareMapTableForm(args.projected, args.substep);
	return createConfigForm(schemas, args.getValue, args.setValue, {
		...options,
		...(args.onSubmit !== undefined ? { onSubmit: args.onSubmit } : {})
	});
}
