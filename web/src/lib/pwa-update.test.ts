// Checkpoint fix (13-04-PLAN.md Task 4, defect 1) — unit coverage over
// scheduleUpdateChecks. Fake timers/event targets (this project's vitest
// config runs environment: 'node', no jsdom — see web/vite.config.ts's own
// comment on why) rather than real window/document, matching the
// dependency-injection shape pwa-update.ts's own doc comment describes.
import { describe, it, expect, vi } from 'vitest';
import { scheduleUpdateChecks, UPDATE_CHECK_INTERVAL_MS } from './pwa-update';
import type {
	UpdateCheckDocumentTarget,
	UpdateCheckEventTarget,
	UpdateCheckTimers
} from './pwa-update';

/** A minimal fake EventTarget that records listeners by type, so a test
 * can both dispatch (call every listener for a type) and assert on
 * exactly which function reference was added/removed. */
function fakeEventTarget(): UpdateCheckEventTarget & {
	listeners: Map<string, Set<() => void>>;
	dispatch: (type: string) => void;
} {
	const listeners = new Map<string, Set<() => void>>();
	return {
		listeners,
		addEventListener(type, listener) {
			if (!listeners.has(type)) listeners.set(type, new Set());
			listeners.get(type)!.add(listener);
		},
		removeEventListener(type, listener) {
			listeners.get(type)?.delete(listener);
		},
		dispatch(type) {
			for (const listener of listeners.get(type) ?? []) listener();
		}
	};
}

function fakeDocumentTarget(
	initialVisibility: string
): UpdateCheckDocumentTarget & { listeners: Map<string, Set<() => void>>; dispatch: (type: string) => void } {
	const base = fakeEventTarget();
	let visibilityState = initialVisibility;
	return {
		...base,
		get visibilityState() {
			return visibilityState;
		},
		set visibilityState(v: string) {
			visibilityState = v;
		}
	};
}

function fakeTimers(): UpdateCheckTimers & {
	handlers: Map<ReturnType<typeof setInterval>, () => void>;
	nextId: number;
} {
	const handlers = new Map<ReturnType<typeof setInterval>, () => void>();
	let nextId = 1;
	return {
		handlers,
		get nextId() {
			return nextId;
		},
		setInterval(handler) {
			const id = nextId++ as unknown as ReturnType<typeof setInterval>;
			handlers.set(id, handler);
			return id;
		},
		clearInterval(id) {
			handlers.delete(id);
		}
	};
}

describe('scheduleUpdateChecks', () => {
	it('sets up a periodic interval at the default UPDATE_CHECK_INTERVAL_MS', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const timers = fakeTimers();
		const setIntervalSpy = vi.fn(timers.setInterval.bind(timers));

		scheduleUpdateChecks(registration, {
			windowTarget: fakeEventTarget(),
			documentTarget: fakeDocumentTarget('visible'),
			timers: { setInterval: setIntervalSpy, clearInterval: timers.clearInterval }
		});

		expect(setIntervalSpy).toHaveBeenCalledTimes(1);
		expect(setIntervalSpy.mock.calls[0][1]).toBe(UPDATE_CHECK_INTERVAL_MS);
	});

	it('respects an explicit intervalMs override', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const timers = fakeTimers();
		const setIntervalSpy = vi.fn(timers.setInterval.bind(timers));

		scheduleUpdateChecks(registration, {
			intervalMs: 5000,
			windowTarget: fakeEventTarget(),
			documentTarget: fakeDocumentTarget('visible'),
			timers: { setInterval: setIntervalSpy, clearInterval: timers.clearInterval }
		});

		expect(setIntervalSpy.mock.calls[0][1]).toBe(5000);
	});

	it('calls registration.update() when the interval fires', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const timers = fakeTimers();

		scheduleUpdateChecks(registration, {
			windowTarget: fakeEventTarget(),
			documentTarget: fakeDocumentTarget('visible'),
			timers
		});

		for (const handler of timers.handlers.values()) handler();

		expect(registration.update).toHaveBeenCalledTimes(1);
	});

	it('calls registration.update() on window focus', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const windowTarget = fakeEventTarget();

		scheduleUpdateChecks(registration, {
			windowTarget,
			documentTarget: fakeDocumentTarget('visible'),
			timers: fakeTimers()
		});

		windowTarget.dispatch('focus');

		expect(registration.update).toHaveBeenCalledTimes(1);
	});

	it('calls registration.update() on window online', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const windowTarget = fakeEventTarget();

		scheduleUpdateChecks(registration, {
			windowTarget,
			documentTarget: fakeDocumentTarget('visible'),
			timers: fakeTimers()
		});

		windowTarget.dispatch('online');

		expect(registration.update).toHaveBeenCalledTimes(1);
	});

	it('calls registration.update() on visibilitychange only when the document becomes visible', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const documentTarget = fakeDocumentTarget('hidden');

		scheduleUpdateChecks(registration, {
			windowTarget: fakeEventTarget(),
			documentTarget,
			timers: fakeTimers()
		});

		// Still hidden — visibilitychange can fire for a hidden->hidden-ish
		// transition in some browsers; must not check while hidden.
		documentTarget.dispatch('visibilitychange');
		expect(registration.update).not.toHaveBeenCalled();

		documentTarget.visibilityState = 'visible';
		documentTarget.dispatch('visibilitychange');
		expect(registration.update).toHaveBeenCalledTimes(1);
	});

	it('the returned cleanup function clears the interval and removes every listener it added', () => {
		const registration = { update: vi.fn().mockResolvedValue(undefined) };
		const windowTarget = fakeEventTarget();
		const documentTarget = fakeDocumentTarget('visible');
		const timers = fakeTimers();

		const cleanup = scheduleUpdateChecks(registration, {
			windowTarget,
			documentTarget,
			timers
		});

		expect(timers.handlers.size, 'expected the interval to be registered before cleanup').toBe(1);
		expect(windowTarget.listeners.get('focus')?.size).toBe(1);
		expect(windowTarget.listeners.get('online')?.size).toBe(1);
		expect(documentTarget.listeners.get('visibilitychange')?.size).toBe(1);

		cleanup();

		expect(timers.handlers.size, 'expected clearInterval to have removed the interval').toBe(0);
		expect(windowTarget.listeners.get('focus')?.size ?? 0).toBe(0);
		expect(windowTarget.listeners.get('online')?.size ?? 0).toBe(0);
		expect(documentTarget.listeners.get('visibilitychange')?.size ?? 0).toBe(0);

		// Dispatching after cleanup must reach nothing.
		windowTarget.dispatch('focus');
		windowTarget.dispatch('online');
		documentTarget.dispatch('visibilitychange');
		for (const handler of timers.handlers.values()) handler();
		expect(registration.update).not.toHaveBeenCalled();
	});
});
