// Temp corpus directories for specs that need real on-disk scratch trees
// (today: the tamper fixtures; originally the filesystem-source specs,
// which moved out with their plugin).
//
// D-03 requires a spec's on-disk corpus to exist BEFORE its kernel boots,
// which is why every filesystem spec seeds one at module scope rather than
// inside a hook. That placement also fixes the corpus's LIFETIME, and this
// is the part that is easy to get wrong: module scope runs once per Node
// PROCESS, whereas `test.afterAll` runs once per Playwright JOB. Those are
// not the same unit, and pairing them cost a CI failure and a permanent
// leak (debug session .planning/debug/resolved/e2e-fs-recursion-ci-flake.md):
//
//  1. `fullyParallel: true` makes Playwright's createTestGroups split a
//     file's tests into groups of `ceil(tests / workers)`, so at workers>=2
//     a two-test file becomes two jobs. afterAll fires at the end of EVERY
//     job, and the dispatcher prefers handing the next job back to the same
//     worker (its workerHash already matches) — where the module cache is
//     warm and module scope will NOT re-run. Job 2 therefore ran against a
//     corpus that job 1's afterAll had already deleted. Because `kernel` is
//     worker-scoped it survived too, still serving the deleted item out of
//     its index, so the first thing to notice anything was wrong was an
//     ENOENT on a plain filesystem call. Sibling specs that only read the
//     stream noticed nothing at all and passed vacuously.
//  2. Playwright loads every spec file in a collection process to build the
//     test tree. Module scope runs there too — but no test and therefore no
//     afterAll ever runs in it, so that process's corpus was orphaned on
//     every single run, green ones included.
//
// The corpus is per-process state, so it must be torn down per process.
// `process.on('exit')` is the only Node hook with that lifetime. Its handler
// must be synchronous, which `rmSync` is. A SIGKILLed process still leaks —
// exactly as it did before, so this is not a regression.
//
// Specs must use this rather than calling `mkdtempSync` themselves;
// e2e/specs/spec-hygiene.spec.ts enforces that mechanically, because
// nothing about a run on a developer's machine reveals the mistake.
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const corpora: string[] = [];
let exitHookInstalled = false;

/**
 * mkdtempCorpus creates a temp corpus directory under the OS temp dir and
 * registers it for removal when THIS Node process exits — matching the
 * lifetime of the module-scope code that calls it.
 *
 * `prefix` is the mkdtemp prefix (Node appends six random characters), e.g.
 * `'topos-e2e-fs-recursion-'`.
 */
export function mkdtempCorpus(prefix: string): string {
	const dir = mkdtempSync(join(tmpdir(), prefix));
	corpora.push(dir);

	// One handler for the whole process, not one per corpus: a spec tree
	// this size would otherwise trip Node's max-listeners warning, and the
	// removal order across corpora is irrelevant since they are siblings.
	if (!exitHookInstalled) {
		exitHookInstalled = true;
		process.on('exit', () => {
			for (const corpus of corpora) {
				rmSync(corpus, { recursive: true, force: true });
			}
		});
	}

	return dir;
}
