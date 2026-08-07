// Tracer checkpoint fix (07-01-PLAN.md Task 1, deviation Rule 2): the
// operator's real config.toml already carried three hand-authored keys the
// Config struct doesn't model — a leftover typo from the Phase 5 migration
// (`[webspaces.cars.silverbullet]` instead of
// `[webspaces.cars.match.silverbullet]`, same for `.proton`/`.signal`).
// GET /api/config was already computing and returning this as `unknown_keys`
// (kernel/httpapi/config.go's toConfigResponse), but nothing in the UI ever
// read it — so config.Store.Save's unknown-key guard (D-01's
// lossless-rewrite prohibition) silently blocked EVERY save through the UI,
// and a user clicking "Save as filter" saw only a post-click Alert (or, if
// they didn't notice it, apparently nothing at all). This guard proves
// WebspaceHeader.svelte surfaces that state proactively, before any save is
// attempted, rather than only after a rejected write.
//
// House pattern (matches source-chip-pill.test.ts / source-chip-selected.
// test.ts): comment-stripped source scanning (no component-mount harness in
// this project's vitest config), a found-non-empty-source guard first, and
// one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const headerPath = join(here, 'WebspaceHeader.svelte');
const pagePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');

const rawHeader = readFileSync(headerPath, 'utf-8');
const rawPage = readFileSync(pagePath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedHeader = stripComments(rawHeader);
const strippedPage = stripComments(rawPage);

describe('unknown-config-keys guard: found non-empty comment-stripped sources', () => {
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
	it('+page.svelte', () => {
		expect(strippedPage.length).toBeGreaterThan(0);
	});
});

describe('the page derives unknownConfigKeys from the config response and passes it down', () => {
	it('+page.svelte reads configResponse.unknown_keys, not a hardcoded empty list', () => {
		expect(
			/unknownConfigKeys\s*=\s*\$derived\(\s*configResponse\?\.unknown_keys/.test(strippedPage),
			'expected unknownConfigKeys to be derived from configResponse.unknown_keys — a hardcoded [] would silently defeat this guard even though the kernel already computes the real value'
		).toBe(true);
		expect(
			/\{unknownConfigKeys\}/.test(strippedPage),
			'expected <WebspaceHeader> to receive the unknownConfigKeys prop — deriving the value with nothing consuming it leaves the warning permanently invisible'
		).toBe(true);
	});
});

describe('WebspaceHeader renders a persistent warning independent of any save attempt', () => {
	it('accepts unknownConfigKeys as a required prop', () => {
		expect(
			/unknownConfigKeys:\s*string\[\]/.test(strippedHeader),
			'expected WebspaceHeader.svelte to declare a typed unknownConfigKeys: string[] prop'
		).toBe(true);
	});

	it('gates a destructive Alert on unknownConfigKeys.length, not on filterError', () => {
		expect(
			/\{#if unknownConfigKeys\.length > 0\}/.test(strippedHeader),
			'expected a dedicated {#if unknownConfigKeys.length > 0} gate — folding this into the filterError Alert would only show it after a save is attempted and rejected, defeating the point of surfacing it proactively'
		).toBe(true);
	});

	it('renders the unrecognised key names in the warning body', () => {
		expect(
			/unknownConfigKeys\.join\(/.test(strippedHeader),
			"expected the warning to name the actual unrecognised keys (via .join), not a generic message — the operator needs to know exactly which table to fix by hand"
		).toBe(true);
	});
});
