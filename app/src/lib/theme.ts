/** Shared with server SPA (`web/src/main.js`) so /my and /app stay in sync. */
export const THEME_STORAGE_KEY = 'attesta_theme';

export type Theme = 'light' | 'dark';

export function getPreferredTheme(
	storage: Pick<Storage, 'getItem'> | null = typeof localStorage !== 'undefined'
		? localStorage
		: null,
	matchMedia: ((query: string) => { matches: boolean }) | null = typeof window !==
	'undefined'
		? window.matchMedia.bind(window)
		: null
): Theme {
	try {
		const stored = storage?.getItem(THEME_STORAGE_KEY);
		if (stored === 'light' || stored === 'dark') return stored;
	} catch {
		/* ignore */
	}
	if (matchMedia?.('(prefers-color-scheme: dark)').matches) return 'dark';
	return 'light';
}

/** Shadcn path: toggle `dark` on <html>. Persist for server SPA sync. */
export function applyTheme(
	theme: Theme,
	root: HTMLElement = document.documentElement,
	storage: Pick<Storage, 'setItem'> | null = typeof localStorage !== 'undefined'
		? localStorage
		: null
): void {
	root.classList.toggle('dark', theme === 'dark');
	try {
		storage?.setItem(THEME_STORAGE_KEY, theme);
	} catch {
		/* ignore */
	}
}

export function toggleTheme(
	root: HTMLElement = document.documentElement,
	storage: Pick<Storage, 'getItem' | 'setItem'> | null = typeof localStorage !==
	'undefined'
		? localStorage
		: null
): Theme {
	const next: Theme = root.classList.contains('dark') ? 'light' : 'dark';
	applyTheme(next, root, storage);
	return next;
}

export function bootTheme(
	root: HTMLElement = document.documentElement,
	storage: Pick<Storage, 'getItem' | 'setItem'> | null = typeof localStorage !==
	'undefined'
		? localStorage
		: null,
	matchMedia: ((query: string) => { matches: boolean }) | null = typeof window !==
	'undefined'
		? window.matchMedia.bind(window)
		: null
): Theme {
	const theme = getPreferredTheme(storage, matchMedia);
	applyTheme(theme, root, storage);
	return theme;
}
