<script lang="ts">
	import { BasicForm, handlers, type FormState } from '@sjsf/form';
	import { setThemeContext } from '@sjsf/shadcn4-theme';
	import * as components from '@sjsf/shadcn4-theme/new-york';
	import { untrack } from 'svelte';
	import { setChromeContext } from './chrome-context.js';
	import { setConfigFormContext } from './config-form-context.js';
	import type { CreateConfigFormResult } from './create-config-form.svelte.js';

	type Props = {
		form: FormState<any>;
		class?: string;
	};

	let { form, class: className }: Props = $props();

	setThemeContext({ components });

	// setContext must run at init; parent remounts via projection key when leftovers change.
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
