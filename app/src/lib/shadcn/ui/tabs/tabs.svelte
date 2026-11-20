<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";
	import { cn } from "#lib/shadcn/utils.js";

	let {
		ref = $bindable(null),
		value = $bindable(""),
		class: className,
		...restProps
	}: TabsPrimitive.RootProps = $props();
</script>

<TabsPrimitive.Root
	bind:ref
	bind:value
	data-slot="tabs"
	class={cn(
		"group/tabs flex w-full min-w-0 flex-col gap-2 data-[orientation=vertical]:flex-row",
		className
	)}
	{...restProps}
/>
