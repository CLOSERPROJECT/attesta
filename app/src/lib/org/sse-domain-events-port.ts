import type { DomainEventsPort } from './domain-events-port.js';

export type SseDomainEventsPortOptions = {
	baseUrl?: string;
	/** Default: globalThis.EventSource; when missing, subscribe is a no-op. */
	EventSource?: typeof EventSource;
	setTimeout?: (fn: () => void, ms: number) => unknown;
	clearTimeout?: (id: unknown) => void;
	/** Reopen delay after the browser gives up on the stream (e.g. 401/404). */
	minRetryMs?: number;
	maxRetryMs?: number;
};

export function createSseDomainEventsPort(
	options: SseDomainEventsPortOptions = {}
): DomainEventsPort {
	const base = options.baseUrl ?? '';
	const EventSourceCtor =
		'EventSource' in options
			? options.EventSource
			: typeof globalThis.EventSource !== 'undefined'
				? globalThis.EventSource
				: undefined;
	const setTimeoutFn = options.setTimeout ?? ((fn: () => void, ms: number) => setTimeout(fn, ms));
	const clearTimeoutFn =
		options.clearTimeout ??
		((id: unknown) => {
			clearTimeout(id as ReturnType<typeof setTimeout>);
		});
	const minRetryMs = options.minRetryMs ?? 5_000;
	const maxRetryMs = options.maxRetryMs ?? 60_000;

	return {
		subscribe(stream, onRevision) {
			if (!EventSourceCtor) {
				return () => {};
			}

			let disposed = false;
			let source: EventSource | null = null;
			let retryMs = minRetryMs;
			let reopenTimer: unknown = null;

			const url = `${base}/api/org/events?stream=${encodeURIComponent(stream)}`;

			function clearReopen() {
				if (reopenTimer != null) {
					clearTimeoutFn(reopenTimer);
					reopenTimer = null;
				}
			}

			function open() {
				if (disposed) return;
				clearReopen();
				const next = new EventSourceCtor!(url, { withCredentials: true });
				source = next;

				next.addEventListener('domain-revision', (e: Event) => {
					if (disposed || source !== next) return;
					const raw = (e as MessageEvent).data;
					try {
						const parsed = JSON.parse(String(raw)) as { revision?: unknown };
						const revision = parsed.revision;
						if (typeof revision !== 'number' || !Number.isInteger(revision) || revision < 0) {
							return;
						}
						retryMs = minRetryMs;
						onRevision(revision);
					} catch {
						/* ignore parse errors */
					}
				});

				next.onerror = () => {
					if (disposed || source !== next) return;
					if (next.readyState !== 2) return;
					next.close();
					if (source === next) source = null;
					const delay = retryMs;
					retryMs = Math.min(retryMs * 2, maxRetryMs);
					reopenTimer = setTimeoutFn(() => {
						reopenTimer = null;
						open();
					}, delay);
				};
			}

			open();

			return () => {
				disposed = true;
				clearReopen();
				source?.close();
				source = null;
			};
		}
	};
}
