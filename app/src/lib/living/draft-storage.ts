export type StorageLike = {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
	removeItem(key: string): void;
	keys(): string[];
};

export const DRAFT_STORAGE_PREFIX = 'attesta:living-draft:';

export function livingDraftStoragePrefix(org: string, userId: string): string {
	return `${DRAFT_STORAGE_PREFIX}${org}:${userId}:`;
}

export function livingDraftStorageKey(org: string, userId: string, draftKey: string): string {
	return `${livingDraftStoragePrefix(org, userId)}${draftKey}`;
}

export function memoryStorage(seed?: Record<string, string>): StorageLike {
	const map = new Map<string, string>(Object.entries(seed ?? {}));
	return {
		getItem(key: string) {
			return map.get(key) ?? null;
		},
		setItem(key: string, value: string) {
			map.set(key, value);
		},
		removeItem(key: string) {
			map.delete(key);
		},
		keys() {
			return [...map.keys()];
		}
	};
}

export function browserStorage(): StorageLike | null {
	try {
		if (typeof localStorage === 'undefined') {
			return null;
		}
		const ls = localStorage;
		const probe = `${DRAFT_STORAGE_PREFIX}__probe__`;
		ls.setItem(probe, '1');
		ls.removeItem(probe);
		return {
			getItem(key: string) {
				try {
					return ls.getItem(key);
				} catch {
					return null;
				}
			},
			setItem(key: string, value: string) {
				try {
					ls.setItem(key, value);
				} catch {
					// private mode / quota
				}
			},
			removeItem(key: string) {
				try {
					ls.removeItem(key);
				} catch {
					// private mode
				}
			},
			keys() {
				try {
					const out: string[] = [];
					for (let i = 0; i < ls.length; i++) {
						const key = ls.key(i);
						if (key != null) {
							out.push(key);
						}
					}
					return out;
				} catch {
					return [];
				}
			}
		};
	} catch {
		return null;
	}
}

export function readStoredDraft(
	storage: StorageLike | null | undefined,
	storageKey: string
): Record<string, unknown> | null {
	if (!storage) {
		return null;
	}
	const raw = storage.getItem(storageKey);
	if (raw === null) {
		return null;
	}
	try {
		const parsed: unknown = JSON.parse(raw);
		if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
			return null;
		}
		return parsed as Record<string, unknown>;
	} catch {
		return null;
	}
}

export function writeStoredDraft(
	storage: StorageLike | null | undefined,
	storageKey: string,
	value: Record<string, unknown>
): void {
	if (!storage) {
		return;
	}
	storage.setItem(storageKey, JSON.stringify(value));
}

export function removeStoredDraft(
	storage: StorageLike | null | undefined,
	storageKey: string
): void {
	if (!storage) {
		return;
	}
	storage.removeItem(storageKey);
}

export function loadDirtyDraftsForIdentity(
	storage: StorageLike | null | undefined,
	org: string,
	userId: string
): Record<string, Record<string, unknown>> {
	const out: Record<string, Record<string, unknown>> = {};
	if (!storage) {
		return out;
	}
	const prefix = livingDraftStoragePrefix(org, userId);
	for (const key of storage.keys()) {
		if (!key.startsWith(prefix)) {
			continue;
		}
		const draftKey = key.slice(prefix.length);
		if (!draftKey) {
			continue;
		}
		const value = readStoredDraft(storage, key);
		if (value === null) {
			continue;
		}
		out[draftKey] = value;
	}
	return out;
}
