# API Coverage — Phase 11 (External Plugins & the Trust Boundary)

No external API integration: this phase extends topos's own kernel (plugin discovery, subprocess launch, config) and its own SvelteKit SPA — the only cross-process contract it touches is topos's in-repo `topos.v1` gRPC plugin contract, which the project itself defines and versions, not a third-party API, SDK, or service.

The deterministic detector (`gsd-core/bin/lib/api-coverage.cjs`) is not present in this installation, so this declaration was made by re-reading the phase scope (ROADMAP Phase 11 section, `11-CONTEXT.md`, `11-RESEARCH.md`, and the six plan bodies). RESEARCH.md's Standard Stack independently records that this phase introduces **no new third-party dependency of any kind** — every capability is Go stdlib plus packages already in `go.mod`.

No capability matrix is produced, because there is no external capability surface to enumerate or subtract from.
