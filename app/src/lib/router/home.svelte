<script lang="ts">
	import OrgShell from '../org/org-shell.svelte';
	import { orgRouteCoordsFromPath } from '../org/nav-place.js';
	import { route } from './index.js';

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
</script>

<OrgShell stream={coords.stream} instance={coords.instance} />
