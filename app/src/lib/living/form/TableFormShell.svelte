<script lang="ts">
	import type { FailureValidationResult, Schema, UiSchemaRoot } from '@sjsf/form';
	import { onDestroy, untrack } from 'svelte';
	import ConfigForm from './ConfigForm.svelte';
	import { createConfigForm } from './create-config-form.svelte.js';
	import { setLeftoverContext, type LeftoverContext } from './leftover-context.js';

	type Props = {
		schema: Schema;
		uiSchema: UiSchemaRoot;
		leftoverIds: string[];
		columns: { id: string; value: string };
		/** Draft owned by StepStackSession — shell does not persist. */
		value: Record<string, unknown>;
		onValueChange: (next: Record<string, unknown>) => void;
		valueType?: LeftoverContext['valueType'];
		disabled?: boolean;
		onSubmit: (value: Record<string, unknown>) => void | Promise<void>;
		onSubmitError?: (result: FailureValidationResult) => void;
	};

	let props: Props = $props();

	// Parent remounts via {#key projectionKey}; capture mount snapshot once.
	const mount = untrack(() => ({
		schema: props.schema,
		uiSchema: props.uiSchema,
		leftoverIds: props.leftoverIds,
		columns: props.columns,
		valueType: props.valueType ?? 'number',
		onSubmit: props.onSubmit,
		onSubmitError: props.onSubmitError
	}));

	setLeftoverContext({
		ids: mount.leftoverIds,
		columns: mount.columns,
		valueType: mount.valueType
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
