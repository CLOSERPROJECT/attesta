<script lang="ts">
	import type { FailureValidationResult, Schema, UiSchemaRoot } from '@sjsf/form';
	import { onDestroy, untrack } from 'svelte';
	import ConfigForm from './ConfigForm.svelte';
	import { createConfigForm } from './create-config-form.svelte.js';
	import { setRowContext } from './row-context.js';

	type Props = {
		schema: Schema;
		uiSchema: UiSchemaRoot;
		rowIds: string[];
		columns: { id: string; value: string };
		/** From TaskFormBundle — Go already classified; default progress for row-table. */
		valueType?: 'number' | 'progress';
		progressEnums?: string[];
		/** Draft owned by StepStackSession — shell does not persist. */
		value: Record<string, unknown>;
		onValueChange: (next: Record<string, unknown>) => void;
		disabled?: boolean;
		onSubmit: (value: Record<string, unknown>) => void | Promise<void>;
		onSubmitError?: (result: FailureValidationResult) => void;
	};

	let props: Props = $props();

	// Parent remounts via {#key projectionKey}; capture mount snapshot once.
	const mount = untrack(() => {
		return {
			schema: props.schema,
			uiSchema: props.uiSchema,
			rowIds: props.rowIds,
			columns: props.columns,
			valueType: props.valueType ?? 'progress',
			progressEnums: props.progressEnums ?? [],
			onSubmit: props.onSubmit,
			onSubmitError: props.onSubmitError
		};
	});

	setRowContext({
		ids: mount.rowIds,
		columns: mount.columns,
		valueType: mount.valueType,
		progressEnums: mount.progressEnums
	});

	const form = createConfigForm(
		{ schema: mount.schema, uiSchema: mount.uiSchema },
		() => props.value,
		(next) => {
			if (props.disabled) return;
			props.onValueChange(next);
		},
		{
			onSubmit: (value) => {
				if (props.disabled) return;
				return mount.onSubmit(value);
			},
			onSubmitError: (result) => {
				mount.onSubmitError?.(result);
			}
		}
	);

	onDestroy(() => {
		form.disposeLocalDraftSync?.();
	});
</script>

<ConfigForm {form} />
