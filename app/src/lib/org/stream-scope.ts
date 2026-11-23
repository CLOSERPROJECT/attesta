/** Org API stream + optional stream-instance query scope. */
export type StreamScope = {
	stream: string;
	/** Omit / empty for Play and cold qr-resolve. */
	instance?: string;
};

/** Append stream (+ optional instance) query params for /api/org/* ports. */
export function appendStreamScope(
	params: URLSearchParams,
	scope: StreamScope
): URLSearchParams {
	params.set('stream', scope.stream);
	const instance = scope.instance?.trim();
	if (instance) {
		params.set('instance', instance);
	}
	return params;
}

/** Build `?stream=…&instance=…` (instance omitted when empty). */
export function streamScopeQuery(scope: StreamScope): string {
	return appendStreamScope(new URLSearchParams(), scope).toString();
}
