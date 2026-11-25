import { afterEach, describe, expect, test, vi } from 'vitest';

import { createSseDomainEventsPort } from './sse-domain-events-port.js';

type Listener = (event: MessageEvent) => void;

class FakeEventSource {
	static instances: FakeEventSource[] = [];
	static CONNECTING = 0;
	static OPEN = 1;
	static CLOSED = 2;

	readonly url: string;
	readonly withCredentials: boolean;
	readyState = FakeEventSource.CONNECTING;
	onerror: ((ev: Event) => void) | null = null;
	readonly #listeners = new Map<string, Set<Listener>>();
	closed = false;

	constructor(url: string, init?: EventSourceInit) {
		this.url = url;
		this.withCredentials = Boolean(init?.withCredentials);
		FakeEventSource.instances.push(this);
		this.readyState = FakeEventSource.OPEN;
	}

	addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
		const fn =
			typeof listener === 'function' ? (listener as Listener) : listener.handleEvent.bind(listener);
		let set = this.#listeners.get(type);
		if (!set) {
			set = new Set();
			this.#listeners.set(type, set);
		}
		set.add(fn as Listener);
	}

	emit(type: string, data: string) {
		const event = { data } as MessageEvent;
		for (const fn of this.#listeners.get(type) ?? []) {
			fn(event);
		}
	}

	fail(readyState: number) {
		this.readyState = readyState;
		this.onerror?.(new Event('error'));
	}

	close() {
		this.closed = true;
		this.readyState = FakeEventSource.CLOSED;
	}
}

describe('createSseDomainEventsPort', () => {
	afterEach(() => {
		vi.useRealTimers();
		FakeEventSource.instances = [];
	});

	test('subscribe encodes the stream and forwards valid domain-revision events', () => {
		const onRevision = vi.fn();
		const port = createSseDomainEventsPort({
			baseUrl: 'https://example.test',
			EventSource: FakeEventSource as unknown as typeof EventSource
		});

		const unsub = port.subscribe('pv-recovery/a b', onRevision);
		const source = FakeEventSource.instances[0];
		expect(source.url).toBe('https://example.test/api/org/events?stream=pv-recovery%2Fa%20b');
		expect(source.withCredentials).toBe(true);

		source.emit('domain-revision', JSON.stringify({ revision: 3 }));
		expect(onRevision).toHaveBeenCalledWith(3);

		source.emit('domain-revision', 'not-json');
		source.emit('domain-revision', JSON.stringify({ revision: -1 }));
		source.emit('domain-revision', JSON.stringify({ revision: 1.5 }));
		source.emit('domain-revision', JSON.stringify({ revision: '2' }));
		expect(onRevision).toHaveBeenCalledTimes(1);

		unsub();
		expect(source.closed).toBe(true);
	});

	test('CONNECTING onerror schedules nothing; CLOSED reopens with backoff', () => {
		vi.useFakeTimers();
		const setTimeoutSpy = vi.fn((fn: () => void, ms: number) => setTimeout(fn, ms));
		const clearTimeoutSpy = vi.fn((id: unknown) =>
			clearTimeout(id as ReturnType<typeof setTimeout>)
		);

		const onRevision = vi.fn();
		const port = createSseDomainEventsPort({
			EventSource: FakeEventSource as unknown as typeof EventSource,
			setTimeout: setTimeoutSpy,
			clearTimeout: clearTimeoutSpy
		});

		port.subscribe('pv-recovery', onRevision);
		const first = FakeEventSource.instances[0]!;

		first.fail(FakeEventSource.CONNECTING);
		expect(setTimeoutSpy).not.toHaveBeenCalled();
		expect(FakeEventSource.instances).toHaveLength(1);

		first.fail(FakeEventSource.CLOSED);
		expect(setTimeoutSpy).toHaveBeenCalledTimes(1);
		expect(setTimeoutSpy).toHaveBeenLastCalledWith(expect.any(Function), 5_000);

		vi.advanceTimersByTime(5_000);
		expect(FakeEventSource.instances).toHaveLength(2);

		FakeEventSource.instances[1]!.fail(FakeEventSource.CLOSED);
		expect(setTimeoutSpy).toHaveBeenLastCalledWith(expect.any(Function), 10_000);

		vi.advanceTimersByTime(10_000);
		expect(FakeEventSource.instances).toHaveLength(3);

		for (let i = 0; i < 10; i++) {
			FakeEventSource.instances.at(-1)!.fail(FakeEventSource.CLOSED);
			vi.advanceTimersByTime(60_000);
		}
		const delays = setTimeoutSpy.mock.calls.map((c: [() => void, number]) => c[1]);
		expect(delays.filter((d: number) => d === 60_000).length).toBeGreaterThan(0);
		expect(Math.max(...delays)).toBe(60_000);

		FakeEventSource.instances.at(-1)!.emit('domain-revision', JSON.stringify({ revision: 1 }));
		FakeEventSource.instances.at(-1)!.fail(FakeEventSource.CLOSED);
		expect(setTimeoutSpy).toHaveBeenLastCalledWith(expect.any(Function), 5_000);
	});

	test('unsubscribe closes the source and cancels a pending reopen', () => {
		vi.useFakeTimers();
		const timers = new Map<unknown, () => void>();
		let nextId = 1;
		const setTimeoutFn = vi.fn((fn: () => void, ms: number) => {
			void ms;
			const id = nextId++;
			timers.set(id, fn);
			return id;
		});
		const clearTimeoutFn = vi.fn((id: unknown) => {
			timers.delete(id);
		});

		const port = createSseDomainEventsPort({
			EventSource: FakeEventSource as unknown as typeof EventSource,
			setTimeout: setTimeoutFn,
			clearTimeout: clearTimeoutFn
		});

		const unsub = port.subscribe('pv-recovery', vi.fn());
		const source = FakeEventSource.instances[0];
		source.fail(FakeEventSource.CLOSED);
		expect(timers.size).toBe(1);

		unsub();
		expect(source.closed).toBe(true);
		expect(clearTimeoutFn).toHaveBeenCalled();
		expect(timers.size).toBe(0);

		const before = FakeEventSource.instances.length;
		for (const fn of timers.values()) fn();
		expect(FakeEventSource.instances).toHaveLength(before);
	});

	test('without EventSource, subscribe is a no-op', () => {
		const port = createSseDomainEventsPort({ EventSource: undefined });
		const unsub = port.subscribe('pv-recovery', vi.fn());
		expect(FakeEventSource.instances).toHaveLength(0);
		unsub();
	});
});
