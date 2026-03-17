## Context

pkgproxy uses the Labstack Echo v5 HTTP framework. Currently no handler exists for the root path `/`. Repository configuration is loaded at startup from YAML into a slice of `Repository` structs accessible via the proxy handler. The service is Go stdlib-friendly and avoids unnecessary external dependencies.

## Goals / Non-Goals

**Goals:**
- Serve a human-readable HTML page at `/` listing configured repositories and their mirrors
- Provide copy-paste package manager config snippets (dnf/yum `.repo`, apt `sources.list`, pacman `mirrorlist`)
- Use only Go stdlib (`html/template`, `net/http`) — no new dependencies
- Integrate cleanly into the existing Echo route setup

**Non-Goals:**
- Authentication or access control for the landing page
- Live cache statistics or disk usage metrics
- JavaScript or external CSS — keep the page static and self-contained
- Configuration editing via the UI

## Decisions

### Use `html/template` embedded in the binary

**Decision:** Embed the HTML template using `embed.FS` (Go 1.16+) or inline as a string constant.

**Rationale:** Keeps the binary self-contained (no external template files at runtime). The template is simple enough that an inline constant or single embedded file is maintainable.

**Alternatives considered:**
- External template file at runtime: more fragile (path dependency), no advantage for a single static template.
- Third-party templating library: unnecessary for a single page.

### New `landing.go` file in `pkg/pkgproxy/`

**Decision:** Add a `LandingHandler` function in a new `pkg/pkgproxy/landing.go` file rather than in `cmd/`.

**Rationale:** Keeps HTTP handler logic co-located with other proxy handlers. The `cmd/` layer remains thin (only route registration).

### Pass repository config directly to the handler

**Decision:** The handler receives the `[]Repository` slice (already loaded at startup) as a closure or struct field.

**Rationale:** Avoids a separate config lookup; repositories are immutable after startup so no locking is needed.

## Risks / Trade-offs

- [Template injection] HTML template uses `html/template` which auto-escapes values — mirror URLs and repo names from config are safely rendered.
- [Large config] Pages with many repositories/mirrors could be verbose — acceptable for an admin-facing page; no pagination needed initially.

## Migration Plan

1. Add `LandingHandler` in `pkg/pkgproxy/landing.go`
2. Register `GET /` route in `cmd/serve.go`
3. No rollback concerns — adding a new route has no impact on existing proxy behavior
