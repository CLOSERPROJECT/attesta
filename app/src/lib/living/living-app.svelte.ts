import { browserStorage } from './draft-storage.js';
import { LivingSession } from './session.svelte.js';

/** Shell hydrates on mount (browser); module ctor skips auto-hydrate. */
export const living = new LivingSession({
	hydrate: false,
	storage: browserStorage()
});