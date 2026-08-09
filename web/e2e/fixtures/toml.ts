// D-04's mechanism: every downstream config-effect assertion reads the
// temp config.toml back as PARSED TOML, never `toContain` on the file's
// raw text — a canonical-writer regression (reordered keys, changed
// quoting, a dropped table) must be caught by structural comparison, not
// masked by a substring that happens to survive.
import { readFileSync } from 'node:fs';
import { parse } from 'smol-toml';

/** readConfigToml reads configPath and returns smol-toml's parse() result. */
export function readConfigToml(configPath: string): Record<string, unknown> {
	const raw = readFileSync(configPath, 'utf-8');
	return parse(raw) as Record<string, unknown>;
}
