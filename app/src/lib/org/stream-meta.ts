/** Title for `/app` chrome from GET /api/org/streams (key match; falls back to key). */
export async function fetchStreamTitle(
	stream: string,
	fetchFn: typeof fetch = fetch
): Promise<string> {
	const key = stream.trim();
	if (!key) return '';
	try {
		const res = await fetchFn('/api/org/streams', {
			credentials: 'include',
			headers: { Accept: 'application/json' },
			cache: 'no-store'
		});
		if (!res.ok) return key;
		const items = (await res.json()) as Array<{ key?: string; title?: string }>;
		const row = items.find((item) => item.key === key);
		const title = row?.title?.trim();
		return title || key;
	} catch {
		return key;
	}
}
