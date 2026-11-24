import { useDatePickerForDateFormat } from '@sjsf/form/fields/extra-widgets/date-picker';
import { overrideByRecord } from '@sjsf/form/lib/resolver';
import { theme as shadcnTheme } from '@sjsf/shadcn4-theme';
// compat resolver maps string enums → enumField; include registers the component.
import '@sjsf/form/fields/extra/enum-include';
import '#lib/shadcn/extra-widgets.js';
import { authoredFieldThemeOverrides } from './authored-field-registry.js';
import CheckboxOnlyField from './checkbox-only-field.svelte';
import CompactArrayItemField from './compact-array-item-field.svelte';
import DerivedLeafField from './derived-leaf-field.svelte';
import FormThemeButton from './form-theme-button.svelte';
import MapTableField from './map-table-field.svelte';
import DataReportField from './data-report-field.svelte';
import ReadonlyField from './readonly-field.svelte';
import HiddenField from './hidden-field.svelte';
import RadioTilesWidget from './radio-tiles-widget.svelte';
import OrgFormLayout from './org-form-layout.svelte';

// override (not extend): base shadcn already defines `button`; extend would never win.
// dataReport / readonly / compactArrayItem / hidden / radioTilesWidget are not
// AUTHORED_FIELD_REGISTRY — register beside button (authoredFieldThemeOverrides drops unknown keys).
// layout: own object-properties (plain div); FieldGroup flex fights form-cols.
const withAuthored = overrideByRecord(shadcnTheme, {
	...authoredFieldThemeOverrides({
		mapTableField: MapTableField,
		derivedLeafField: DerivedLeafField,
		checkboxOnlyField: CheckboxOnlyField
	}),
	dataReportField: DataReportField,
	readonlyField: ReadonlyField,
	hiddenField: HiddenField,
	compactArrayItemField: CompactArrayItemField,
	radioTilesWidget: RadioTilesWidget,
	button: FormThemeButton,
	layout: OrgFormLayout
});

export const theme = useDatePickerForDateFormat(withAuthored);
