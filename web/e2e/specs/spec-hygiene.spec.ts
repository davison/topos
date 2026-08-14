// A structural gate over the spec tree itself, not over the product.
//
// It exists because the defect it pins (debug session
// .planning/debug/resolved/e2e-fs-recursion-ci-flake.md) is invisible to
// every other gate: a spec that seeds module-scope state and tears it down
// in `test.afterAll` passes on a developer's machine essentially always
// (12 cores -> 6 workers -> the file's tests land in separate worker
// PROCESSES, each of which re-runs module scope and owns its own state),
// and fails only when Playwright's dispatcher hands two jobs of the same
// file back to one worker — the arrangement a 2-worker CI runner produces
// routinely. Worse, only a spec that touches the state ON DISK fails; a
// spec that merely reads the stream keeps passing against the worker-scoped
// kernel's stale index and reports green while proving nothing.
//
// So there is no runtime assertion that can catch this class reliably. A
// structural one can, and does so on every run.
//
// This spec boots nothing: it never requests the `kernel` fixture, so no
// kernel process, plugin binary or temp directory is created by it.
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

import { test, expect } from '../fixtures/kernel';

/** This file's own name — exempt from the rules it enforces. */
const SELF = 'spec-hygiene.spec.ts';

function specFiles(specsDir: string): string[] {
	return readdirSync(specsDir)
		.filter((name) => name.endsWith('.spec.ts'))
		.filter((name) => name !== SELF)
		.sort();
}

/**
 * Returns the source text of every `test.afterAll(...)` call in `source`,
 * extracted by scanning for the balanced closing parenthesis rather than by
 * regex — a hook body containing its own parentheses (every one of them
 * does) would defeat a non-greedy pattern and silently match nothing, which
 * is the failure mode that makes a lint-shaped test worthless.
 */
function afterAllBodies(source: string): string[] {
	const bodies: string[] = [];
	const marker = 'test.afterAll(';
	let from = 0;

	for (;;) {
		const start = source.indexOf(marker, from);
		if (start === -1) return bodies;

		let depth = 0;
		let end = -1;
		for (let i = start + marker.length - 1; i < source.length; i++) {
			const ch = source[i];
			if (ch === '(') depth++;
			else if (ch === ')') {
				depth--;
				if (depth === 0) {
					end = i;
					break;
				}
			}
		}
		if (end === -1) {
			// Unbalanced source: report the remainder so the assertion below
			// fails loudly rather than skipping the file.
			bodies.push(source.slice(start));
			return bodies;
		}

		bodies.push(source.slice(start, end + 1));
		from = end + 1;
	}
}

test.describe('e2e spec-tree hygiene', () => {
	test('no spec creates a temp corpus directly — fixtures/corpus.ts owns that lifetime', () => {
		const specsDir = dirname(test.info().file);
		const offenders = specFiles(specsDir).filter((name) =>
			readFileSync(join(specsDir, name), 'utf8').includes('mkdtempSync')
		);

		expect(
			offenders,
			`These specs call mkdtempSync directly: ${offenders.join(', ')}.\n` +
				'Module scope runs once per Node PROCESS (including Playwright’s spec-collection\n' +
				'process, which runs no test at all), so a corpus created there can only be removed\n' +
				'on process exit. Use mkdtempCorpus() from ../fixtures/corpus, which registers that\n' +
				'cleanup for you.'
		).toEqual([]);
	});

	test('no spec removes a directory in test.afterAll — that hook runs once per JOB, not once per process', () => {
		const specsDir = dirname(test.info().file);
		const offenders = specFiles(specsDir).filter((name) =>
			afterAllBodies(readFileSync(join(specsDir, name), 'utf8')).some((body) =>
				body.includes('rmSync')
			)
		);

		expect(
			offenders,
			`These specs remove a directory inside test.afterAll: ${offenders.join(', ')}.\n` +
				'With fullyParallel, Playwright splits a file into ceil(tests / workers) groups and\n' +
				'runs afterAll at the end of EVERY group, then prefers handing the next group back to\n' +
				'the same worker — whose module cache is warm, so module-scope state is NOT re-seeded.\n' +
				'The next test then runs against state this hook already deleted. Tie the cleanup to\n' +
				'the process instead (see ../fixtures/corpus.ts).'
		).toEqual([]);
	});
});
