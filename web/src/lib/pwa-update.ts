// Checkpoint fix (13-04-PLAN.md Task 4, defect 1): a long-lived, standalone
// installed-PWA window never navigates, and the browser's own automatic
// ServiceWorker update check fires only on navigation (or roughly every
// 24h in the background) — neither ever happens for a window a user just
// leaves open, so the SPA's existing autoUpdate/onNeedReload wiring
// (+layout.svelte) never gets a chance to fire: it reacts to a new
// ServiceWorker activating, but nothing was ever asking the browser to
// check whether one exists.
//
// scheduleUpdateChecks closes that gap by calling the registration's own
// `update()` method — a plain refetch-and-byte-compare of the SW script
// itself, never an /api request, so this cannot reintroduce the offline
// API caching this milestone's Requirements document rules out — on a
// lightweight schedule: periodically, and on the moments a user is most
// likely to have just come back to a possibly-stale window (the tab/window
// regaining visibility or focus, or the network coming back online after a
// blip). Once `update()` finds a new build, the browser's own SW lifecycle
// (install -> activate, since the generated SW always calls
// self.skipWaiting()) fires the EXACT SAME `activated` event
// +layout.svelte's onNeedReload already handles — this module only ever
// asks the question more often, it never changes what happens once the
// answer is "yes".

/** The minimal registration surface this module needs — matches exactly
 * the one method called here (ServiceWorkerRegistration.update), narrowed
 * so a test can pass a plain mock object instead of a real registration. */
export interface UpdateCheckRegistration {
	// Widened to Promise<unknown> — real ServiceWorkerRegistration.update()
	// types resolve with the registration itself (or void, across lib.dom
	// versions); this module never reads the resolved value, only that a
	// call happened, so callers can pass a real registration OR a narrow
	// test double without either shape fighting the other.
	update: () => Promise<unknown>;
}

/** window and document both satisfy this — the minimal event-target
 * surface scheduleUpdateChecks needs from either. */
export interface UpdateCheckEventTarget {
	addEventListener: (type: string, listener: () => void) => void;
	removeEventListener: (type: string, listener: () => void) => void;
}

/** document additionally exposes visibilityState — read live (not
 * captured at setup time) so the visibilitychange handler always sees the
 * CURRENT value at the moment the event actually fires. */
export interface UpdateCheckDocumentTarget extends UpdateCheckEventTarget {
	// Not `readonly`: the real DOM's document.visibilityState is
	// externally read-only (a getter with no public setter), but marking
	// it readonly HERE would forbid a test fixture's own mutable mock
	// property of the same name — this interface only ever reads the
	// value, so it doesn't need to assert anything about whether a given
	// implementation's own field is externally assignable.
	visibilityState: string;
}

/** Narrowed to exactly the two globalThis.setInterval/clearInterval calls
 * this module makes — a test supplies fakes, production supplies the real
 * globals. */
export interface UpdateCheckTimers {
	setInterval: (handler: () => void, ms: number) => ReturnType<typeof setInterval>;
	clearInterval: (id: ReturnType<typeof setInterval>) => void;
}

export interface ScheduleUpdateChecksOptions {
	/** @default UPDATE_CHECK_INTERVAL_MS */
	intervalMs?: number;
	windowTarget: UpdateCheckEventTarget;
	documentTarget: UpdateCheckDocumentTarget;
	timers: UpdateCheckTimers;
}

/** Default periodic check interval: frequent enough that an installed
 * app left open genuinely self-updates "within a few seconds" of a
 * kernel rebuild+restart landing (the plan's own success criterion),
 * infrequent enough that it is not a meaningful resource cost for a
 * plain SW-script HEAD-shaped fetch. */
export const UPDATE_CHECK_INTERVAL_MS = 20_000;

/**
 * Wires registration.update() to fire periodically and on
 * focus/online/visibilitychange-to-visible. Returns a cleanup function
 * that clears the interval and removes every listener this call
 * installed — callers that care about tearing this down cleanly (tests,
 * a future non-root-layout caller) can call it; the root layout itself
 * does not, since it lives for the SPA's whole session and there is
 * nothing meaningful to tear down before the page itself unloads.
 */
export function scheduleUpdateChecks(
	registration: UpdateCheckRegistration,
	options: ScheduleUpdateChecksOptions
): () => void {
	const intervalMs = options.intervalMs ?? UPDATE_CHECK_INTERVAL_MS;

	const check = () => {
		void registration.update();
	};
	const checkIfVisible = () => {
		if (options.documentTarget.visibilityState === 'visible') check();
	};

	const intervalId = options.timers.setInterval(check, intervalMs);
	options.windowTarget.addEventListener('focus', check);
	options.windowTarget.addEventListener('online', check);
	options.documentTarget.addEventListener('visibilitychange', checkIfVisible);

	return () => {
		options.timers.clearInterval(intervalId);
		options.windowTarget.removeEventListener('focus', check);
		options.windowTarget.removeEventListener('online', check);
		options.documentTarget.removeEventListener('visibilitychange', checkIfVisible);
	};
}
