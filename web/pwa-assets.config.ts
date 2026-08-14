import { defineConfig, minimal2023Preset } from '@vite-pwa/assets-generator/config';

// PWA icon set derivation (13-UI-SPEC.md E7): the entire icon set is
// generated at build time from the app's own existing 1024x1024 source
// icon — no new icon artwork is authored, and nothing here references a
// third-party host. `minimal2023Preset` produces exactly the set the
// manifest needs (192px, 512px, and a maskable variant with correct
// safe-zone padding, tool-handled rather than hand-authored) without
// pulling in the preset's optional apple-touch/splash-screen assets this
// app doesn't need (no iOS-specific install surface is in scope this
// phase).
export default defineConfig({
	preset: minimal2023Preset,
	images: ['static/app-icon.png']
});
