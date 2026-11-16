import { browserStorage } from './draft-storage.js';
import { OrgSession } from './session.svelte.js';

/** Shell hydrates on mount (browser); module ctor skips auto-hydrate. */
export const orgSession = new OrgSession({
	hydrate: false,
	storage: browserStorage()
});