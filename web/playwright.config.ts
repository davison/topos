import { defineConfig, devices } from '@playwright/test';

// D-01/D-02/D-15: this config deliberately does NOT set `use.baseURL` and
// does NOT use Playwright's single-shared-server-launch option. That
// option boots exactly one shared server for the whole run, which is
// structurally incompatible with D-02's fresh-kernel-per-spec-file
// requirement — the kernel fixture (web/e2e/fixtures/kernel.ts) supplies
// a per-file baseURL instead, once each file's own hermetic kernel has
// booted.
export default defineConfig({
	testDir: './e2e/specs',
	fullyParallel: true,
	// D-15: zero retries, everywhere, local and CI. A flaky test is a bug
	// and gets fixed — hermetic per-file kernels remove the usual flake
	// sources, so a retry here would only mask a real regression.
	retries: 0,
	forbidOnly: true,
	outputDir: './e2e/.artifacts/test-results',
	reporter: [['list'], ['html', { outputFolder: './e2e/.artifacts/report', open: 'never' }]],
	use: {
		// D-15: every failure saves a trace and a screenshot for diagnosis.
		// video stays off — traces already carry the full timeline and a
		// video adds artifact weight with no extra diagnostic value here.
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		video: 'off'
	},
	// D-14: Chromium is listed first and is the only engine CI runs
	// (`make e2e`'s E2E_PROJECT defaults to "chromium"). Firefox and
	// WebKit are defined here so a manual cross-engine pass is one flag
	// away (`--project=firefox` / `E2E_PROJECT=firefox`) but neither is
	// ever part of the automated gate.
	projects: [
		{ name: 'chromium', use: { ...devices['Desktop Chrome'] } },
		{ name: 'firefox', use: { ...devices['Desktop Firefox'] } },
		{ name: 'webkit', use: { ...devices['Desktop Safari'] } }
	]
});
