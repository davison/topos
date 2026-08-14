// The app's first toast primitive (13-UI-SPEC.md E3), backing the
// per-item exclude/include undo/failure flow (D-02/D-08) and, later, the
// PWA update notice. Copy here is taken verbatim from 13-UI-SPEC.md's
// Copywriting Contract — this is the ONE place both the single-item
// detail-pane path and the (later) bulk action-bar path get their toast
// copy from, so the two can never drift in wording.
import { toast } from 'svelte-sonner';

/** MarkVerb is the past-tense form the undo toast's body uses. */
export type MarkVerb = 'Excluded' | 'Included';

/** MarkFailureVerb is the infinitive form the write-failure toast's body uses. */
export type MarkFailureVerb = 'exclude' | 'include';

// itemNoun is the ONE place the item/items pluralization decision is
// made — singular for a count of exactly 1, plural otherwise (including
// 0, matching ordinary English usage). Both markPhrase and
// markFailureToast's body route through this, so a count's grammatical
// form can never disagree between the two toast shapes.
function itemNoun(count: number): string {
	return count === 1 ? 'item' : 'items';
}

/**
 * markPhrase returns the pluralized undo-toast body — e.g. "Excluded 1
 * item" / "Excluded 2 items". This is the one place that phrase is
 * built; markSuccessToast is its only caller here, and the later bulk
 * action-bar path reuses this same function rather than reimplementing
 * pluralization.
 */
export function markPhrase(verb: MarkVerb, count: number): string {
	return `${verb} ${count} ${itemNoun(count)}`;
}

// reversedFailureVerb maps an undo toast's own verb to the INFINITIVE verb
// naming the write undo itself performs — undoing an "Excluded" toast
// issues an include write, and vice versa. Used only to pick the correct
// write-failure copy if the undo's own mirror write fails (13-UI-SPEC.md
// E3 backstop: "reuses the existing write-failure toast copy rather than
// a bespoke undo-failed string").
function reversedFailureVerb(verb: MarkVerb): MarkFailureVerb {
	return verb === 'Excluded' ? 'include' : 'exclude';
}

/**
 * markSuccessToast fires immediately on a successful mark write (D-02/
 * D-08): body is markPhrase(verb, count), with a single Undo action that
 * re-issues the mirror write via onUndo. The action disables itself
 * (re-issuing the SAME toast id with a no-op onClick — svelte-sonner
 * upserts a toast in place when called again with an existing id) for
 * the duration of onUndo's promise, so a second click cannot double-fire
 * the reversal. Auto-dismisses after 5000ms if not acted on — the
 * excluded-items view (a later plan) is the durable reversal path after
 * that. If onUndo itself rejects, the write-failure toast fires with the
 * REVERSED verb (undoing an exclude is an include write, and vice
 * versa) — the same copy path markFailureToast always uses, never a
 * bespoke "undo failed" string.
 */
export function markSuccessToast({
	verb,
	count,
	onUndo
}: {
	verb: MarkVerb;
	count: number;
	onUndo: () => Promise<void>;
}): void {
	const body = markPhrase(verb, count);
	const failureVerb = reversedFailureVerb(verb);

	const id = toast(body, {
		duration: 5000,
		action: {
			label: 'Undo',
			onClick: () => {
				// Re-issuing toast() with the SAME id updates this toast in
				// place rather than creating a second one — this disables the
				// action (a no-op onClick) for the duration of the write.
				toast(body, {
					id,
					duration: 5000,
					action: { label: 'Undo', onClick: () => {} }
				});
				void onUndo()
					.then(() => {
						toast.dismiss(id);
					})
					.catch(() => {
						markFailureToast({ verb: failureVerb, count });
					});
			}
		}
	});
}

/**
 * markFailureToast fires the destructive-toned write-failure toast:
 * "Couldn't {verb} {N} item(s) — try again." — matches this app's
 * existing terse "Couldn't {verb} — {next step}" failure-copy convention
 * (OpenInSource.svelte's own "Couldn't open"). No action button; the
 * originating control (detail-pane button or, later, the action bar)
 * stays visible and re-enabled as the retry path.
 */
export function markFailureToast({ verb, count }: { verb: MarkFailureVerb; count: number }): void {
	toast.error(`Couldn't ${verb} ${count} ${itemNoun(count)} — try again.`);
}

/**
 * pwaUpdatedToast fires the informational, neutral-toned toast explaining
 * an unannounced reload (13-UI-SPEC.md E3.3/E8): the ServiceWorker's
 * `autoUpdate` strategy silently activates a new build and reloads the
 * page with no user action, so this toast exists purely to explain what
 * just happened, not to ask for one. No action button — the reload has
 * already happened by the time this renders. Auto-dismisses after 4000ms.
 * Copy is the contract-exact string from the Copywriting Contract; do not
 * reword.
 */
export function pwaUpdatedToast(): void {
	toast('topos updated to the latest version.', { duration: 4000 });
}
