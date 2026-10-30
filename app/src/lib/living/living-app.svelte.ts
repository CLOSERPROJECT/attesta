import { LivingSession } from './session.svelte.js';

/** Shared living session for `/app`. Hydrate from the shell on mount (browser). */
export const living = new LivingSession({ hydrate: false });
