import { useDatePickerForDateFormat } from '@sjsf/form/fields/extra-widgets/date-picker';
import { overrideByRecord } from '@sjsf/form/lib/resolver';
import { theme as shadcnTheme } from '@sjsf/shadcn4-theme';
// compat resolver maps string enums → enumField; include registers the component.
import '@sjsf/form/fields/extra/enum-include';
// format: date → shadcn DatePicker (register Substep).
import '@sjsf/shadcn4-theme/extra-widgets/date-picker-include';
// register idoneitaRiutilizzo: ui:components.selectWidget → radioWidget.
import '@sjsf/shadcn4-theme/extra-widgets/radio-include';
import { authoredFieldThemeOverrides } from './authored-field-registry.js';
import CheckboxOnlyField from './checkbox-only-field.svelte';
import CompactArrayItemField from './compact-array-item-field.svelte';
import DerivedLeafField from './derived-leaf-field.svelte';
import FormThemeButton from './form-theme-button.svelte';
import LeftoverTableField from './leftover-table-field.svelte';
import MapTableField from './map-table-field.svelte';
import DataReportField from './data-report-field.svelte';
import ReadonlyField from './readonly-field.svelte';
import ValuePrintField from './value-print-field.svelte';

// override (not extend): base shadcn already defines `button`; extend would never win.
// valuePrint / dataReport / readonly / compactArrayItem are not AUTHORED_FIELD_REGISTRY —
// register beside button (authoredFieldThemeOverrides drops unknown keys).
const withAuthored = overrideByRecord(shadcnTheme, {
	...authoredFieldThemeOverrides({
		leftoverTableField: LeftoverTableField,
		mapTableField: MapTableField,
		derivedLeafField: DerivedLeafField,
		checkboxOnlyField: CheckboxOnlyField
	}),
	valuePrintField: ValuePrintField,
	dataReportField: DataReportField,
	readonlyField: ReadonlyField,
	compactArrayItemField: CompactArrayItemField,
	button: FormThemeButton
});

export const theme = useDatePickerForDateFormat(withAuthored);
