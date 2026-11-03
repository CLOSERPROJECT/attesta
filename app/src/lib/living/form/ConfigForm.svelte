<script lang="ts">
	import { BasicForm, handlers, type FormState } from '@sjsf/form';
	import { setShadcnContext } from '#lib/shadcn/set-shadcn-context.js';
	import { untrack } from 'svelte';
	import { setChromeContext } from './chrome-context.js';
	import { setConfigFormContext } from './config-form-context.js';
	import type { CreateConfigFormResult } from './create-config-form.svelte.js';

	type Props = {
		form: FormState<any>;
		class?: string;
	};

	let { form, class: className }: Props = $props();

	setShadcnContext();

	// setContext must run at init; parent remounts via projection key when row ids change.
	const mounted = untrack(() => form as CreateConfigFormResult<Record<string, unknown>>);
	const configForm = mounted.configForm;
	if (configForm) {
		setConfigFormContext(configForm.derivedContext());
	}
	const liveTaskForm = mounted.liveTaskForm;
	if (liveTaskForm) {
		setChromeContext(liveTaskForm.chromeContext());
	}
</script>

<BasicForm {form} {@attach handlers(form)} class={className} />
