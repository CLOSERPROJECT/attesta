<script lang="ts">
	import { BasicForm, type FormState } from '@sjsf/form';
	import { setShadcnContext } from '#lib/shadcn/set-shadcn-context.js';
	import { untrack } from 'svelte';
	import { setChromeContext } from './chrome-context.js';
	import {
		setConfigFormContext,
		type ConfigFormDerivedContext
	} from './config-form-context.js';
	import type { CreateConfigFormResult } from './create-config-form.svelte.js';

	type Props = {
		form: FormState<any>;
		class?: string;
	};

	let { form, class: className }: Props = $props();

	setShadcnContext();

	// setContext at init. Resolve engines via LiveTaskForm.form so updatePlan can
	// swap the sjsf instance (derived branch flip) without a `{#key}` remount.
	const mounted = untrack(() => form as CreateConfigFormResult<Record<string, unknown>>);
	const liveTaskForm = mounted.liveTaskForm;
	if (liveTaskForm) {
		setChromeContext(liveTaskForm.chromeContext());
	}

	function activeConfigForm() {
		return liveTaskForm?.form?.configForm ?? mounted.configForm;
	}

	const initialConfig = activeConfigForm();
	if (initialConfig || liveTaskForm) {
		const ctx: ConfigFormDerivedContext = {
			isOverridden: (path) => activeConfigForm()?.isOverridden(path) ?? false,
			isOverridable: (path) => activeConfigForm()?.isOverridable(path) ?? false,
			computed: (path) => activeConfigForm()?.computed(path),
			restore: (path) => {
				activeConfigForm()?.restore(path);
			},
			writePath: (path, value) => {
				activeConfigForm()?.writePath(path, value);
			}
		};
		setConfigFormContext(ctx);
	}
</script>

<!-- Theme owns onsubmit; do not {@attach handlers(form)} (double Commit). -->
<BasicForm {form} class={className} />
