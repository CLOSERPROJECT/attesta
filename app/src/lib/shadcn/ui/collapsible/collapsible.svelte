<script lang="ts">
	import { Collapsible as CollapsiblePrimitive } from "bits-ui";

	let {
		ref = $bindable(null),
		open = $bindable(false),
		onOpenChange,
		...restProps
	}: CollapsiblePrimitive.RootProps = $props();
</script>

<CollapsiblePrimitive.Root
	bind:ref
	bind:open
	{onOpenChange}
	data-slot="collapsible"
	{...restProps}
/>
