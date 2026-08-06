// The app's only site that constructs a real ResizeObserver. Attachment is
// extracted here (rather than inlined in WebspaceHeader.svelte) so it is
// provable by behaviour in the existing environment: 'node' vitest runner,
// with no component-mount harness and no new dependency — see
// 06-04-PLAN.md's decision record. The `createObserver` factory parameter
// is the seam that makes this possible: tests inject a fake observer and
// assert on its recorded calls instead of needing a real DOM.
//
// The real observer ignores a repeated `observe()` of the same element, so
// this module does no de-duplication of its own.

/** The subset of the real ResizeObserver API this app actually uses. */
export interface ResizeObserverLike {
	observe(target: Element): void;
	disconnect(): void;
}

/** Constructs a ResizeObserverLike, wiring its resize callback to `onResize`. */
export type CreateResizeObserver = (onResize: () => void) => ResizeObserverLike;

/**
 * Observes every bound element in `targets`, invoking `onResize` whenever
 * any of them resizes, and returns an idempotent teardown.
 *
 * - If every target is unbound (undefined/null), no observer is
 *   constructed at all and the returned teardown is a safe no-op. This is
 *   the state on first mount, before an async load (e.g. GET
 *   /api/sources) resolves and the elements it gates first bind —
 *   constructing an observer with nothing to watch would be dead wiring.
 * - Otherwise one observer is constructed and each bound target is
 *   observed once, in the order given.
 * - The returned teardown disconnects the observer exactly once; a second
 *   call is inert, since a ref-driven caller (a Svelte $effect) can
 *   legitimately tear down more than once.
 *
 * `createObserver` defaults lazily to the real ResizeObserver constructor
 * — referenced only inside this default, never evaluated at module scope
 * — so importing this module has no DOM dependency and is safe under the
 * test runner's `environment: 'node'`.
 */
export function observeResize(
	targets: readonly (Element | undefined | null)[],
	onResize: () => void,
	createObserver: CreateResizeObserver = (cb) => new ResizeObserver(cb)
): () => void {
	const boundTargets = targets.filter((target): target is Element => target != null);

	if (boundTargets.length === 0) {
		return () => {};
	}

	const observer = createObserver(() => onResize());
	for (const target of boundTargets) {
		observer.observe(target);
	}

	let disconnected = false;
	return () => {
		if (disconnected) return;
		disconnected = true;
		observer.disconnect();
	};
}
