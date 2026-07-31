// Source-scan guard over web/src/lib/components/*.svelte (top level only —
// deliberately NOT recursing into ui/, which holds vendored shadcn-svelte/
// bits-ui primitives this project does not author, whose future date-picker
// components may legitimately use browser locale APIs). No component-mount
// harness exists (web/vite.config.ts's test block: environment: 'node'), so
// this test reads component source text off disk instead of mounting one.
//
// `toLocaleDateString` (the un-pinned browser locale API — the exact
// warning this test guards against, 03-REVIEW.md WR-01) is named here, in
// the scanner, precisely because it must appear nowhere in the components
// being scanned: any first-party component under web/src/lib/components/
// must render an item date through the shared, UTC-pinned formatItemDate
// (web/src/lib/format.ts) instead of a locally declared formatter.

import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const componentsDir = dirname(fileURLToPath(import.meta.url));

function topLevelSvelteFiles(): string[] {
	return readdirSync(componentsDir).filter((name) => name.endsWith('.svelte'));
}

describe('date-format source-scan guard', () => {
	const files = topLevelSvelteFiles();

	it('found at least one top-level component to scan', () => {
		// Guards against a silent no-op: a wrong directory resolution must
		// fail loudly here rather than making every assertion below
		// vacuously pass over an empty file list.
		expect(files.length).toBeGreaterThan(0);
	});

	it('contains no locally-declared, un-pinned date formatter (toLocaleDateString)', () => {
		for (const file of files) {
			const source = readFileSync(join(componentsDir, file), 'utf-8');
			expect(
				source.includes('toLocaleDateString'),
				`${file} calls toLocaleDateString directly — item dates must render through the shared, UTC-pinned formatItemDate from $lib/format instead, so the calendar day can never disagree between surfaces`
			).toBe(false);
		}
	});

	it('imports formatItemDate in every component that renders timestamp_unix', () => {
		for (const file of files) {
			const source = readFileSync(join(componentsDir, file), 'utf-8');
			if (source.includes('timestamp_unix')) {
				expect(
					source.includes('formatItemDate'),
					`${file} references timestamp_unix but does not import formatItemDate from $lib/format — an item date must always render through the shared UTC-pinned formatter`
				).toBe(true);
			}
		}
	});
});
