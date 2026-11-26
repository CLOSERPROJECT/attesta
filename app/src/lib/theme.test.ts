import { describe, expect, test } from 'vitest';

import {
	THEME_STORAGE_KEY,
	applyTheme,
	bootTheme,
	getPreferredTheme,
	toggleTheme
} from './theme.js';

function memoryStorage(initial: Record<string, string> = {}): Storage {
	const map = new Map(Object.entries(initial));
	return {
		get length() {
			return map.size;
		},
		clear() {
			map.clear();
		},
		getItem(key: string) {
			return map.has(key) ? map.get(key)! : null;
		},
		key(index: number) {
			return [...map.keys()][index] ?? null;
		},
		removeItem(key: string) {
			map.delete(key);
		},
		setItem(key: string, value: string) {
			map.set(key, value);
		}
	};
}

describe('theme', () => {
	test('prefers stored attesta_theme over OS', () => {
		const storage = memoryStorage({ [THEME_STORAGE_KEY]: 'dark' });
		expect(
			getPreferredTheme(storage, () => ({ matches: false }))
		).toBe('dark');
		storage.setItem(THEME_STORAGE_KEY, 'light');
		expect(
			getPreferredTheme(storage, () => ({ matches: true }))
		).toBe('light');
	});

	test('falls back to prefers-color-scheme', () => {
		expect(getPreferredTheme(memoryStorage(), () => ({ matches: true }))).toBe(
			'dark'
		);
		expect(getPreferredTheme(memoryStorage(), () => ({ matches: false }))).toBe(
			'light'
		);
	});

	function fakeRoot(initialDark = false): HTMLElement {
		const classes = new Set<string>(initialDark ? ['dark'] : []);
		return {
			classList: {
				contains: (c: string) => classes.has(c),
				toggle: (c: string, force?: boolean) => {
					const on = force ?? !classes.has(c);
					if (on) classes.add(c);
					else classes.delete(c);
					return on;
				},
				add: (c: string) => {
					classes.add(c);
				},
				remove: (c: string) => {
					classes.delete(c);
				}
			}
		} as HTMLElement;
	}

	test('applyTheme toggles html.dark and persists', () => {
		const root = fakeRoot();
		const storage = memoryStorage();
		applyTheme('dark', root, storage);
		expect(root.classList.contains('dark')).toBe(true);
		expect(storage.getItem(THEME_STORAGE_KEY)).toBe('dark');
		applyTheme('light', root, storage);
		expect(root.classList.contains('dark')).toBe(false);
		expect(storage.getItem(THEME_STORAGE_KEY)).toBe('light');
	});

	test('toggleTheme flips class', () => {
		const root = fakeRoot();
		const storage = memoryStorage();
		expect(toggleTheme(root, storage)).toBe('dark');
		expect(root.classList.contains('dark')).toBe(true);
		expect(toggleTheme(root, storage)).toBe('light');
		expect(root.classList.contains('dark')).toBe(false);
	});

	test('bootTheme applies preferred', () => {
		const root = fakeRoot();
		const storage = memoryStorage({ [THEME_STORAGE_KEY]: 'dark' });
		expect(bootTheme(root, storage, () => ({ matches: false }))).toBe('dark');
		expect(root.classList.contains('dark')).toBe(true);
	});
});
