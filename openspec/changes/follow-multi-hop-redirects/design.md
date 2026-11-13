## Context

`tryMirrors` (`pkg/pkgproxy/proxy.go`) tries each mirror up to `retries` times. After the initial upstream request, a single `if` checks for a redirect status, follows `Location` once, and then evaluates the response: 200 wins, a 5xx retries, anything else moves on to the next mirror. Fedora COPR now redirects twice (COPR → Red Hat Pulp → S3), so the second `302` falls through and is returned to the client by the handler as-is.

The handler already turns `(nil, err)` from `tryMirrors` into 502 Bad Gateway, so no new error plumbing is needed.

## Goals / Non-Goals

**Goals:**
- Follow redirect chains of up to 10 hops per mirror attempt.
- Reject HTTPS → HTTP downgrades at any hop.
- Treat an unusable redirect chain as a failed mirror and never leak a 3xx to the client.

**Non-Goals:**
- A configurable hop limit.
- SSRF hardening (blocking private or loopback redirect targets).
- Changing method or header semantics on redirect (e.g. 303 → GET).
- Pointing the `copr` mirror at the Pulp URL directly. The official COPR URL remains the configured entry point.

## Decisions

**Manual loop instead of `http.Client` with `CheckRedirect`.**
Replace the single `if` with a loop bounded by a `maxRedirects = 10` constant. Before each hop, check the hop count and the scheme transition. It's a small diff that keeps today's semantics: same method and buffered body replayed, headers filtered by `allowedRequestHeaders`, and one `upstream response` log line per hop. *Alternative:* `http.Client` provides the 10-hop default and a `CheckRedirect` hook, but it rewrites 301/302/303 to GET, strips headers on cross-host hops, drops per-hop logging, and requires reworking `forwardClientRequestToOrigin` and the transport tests.

**Limit of 10, as a fixed constant.**
This matches Go's `http.Client` default, so the behaviour is familiar. COPR needs 2 hops. *Alternative:* 5 (tighter, but no real benefit) or a per-repo config key (adds docs and validation with no use case).

**Downgrade check compares each hop with the previous URL.**
Reject when the current URL's scheme is `https` and the `Location` scheme is `http`. Because each hop is checked against the one before it, an `https` link anywhere in the chain prevents a later `http` hop. An `http`-configured mirror can still redirect to `http`. The check applies to the first hop too.

**Failure handling reuses the existing connection-error path.**
Exceeding the limit or rejecting a downgrade closes the response, sets `rsp = nil` and a descriptive `err`, logs a warning, and `break`s to the next mirror. These are not retried, just like a bad `Location`. If the last mirror ends this way, `tryMirrors` returns `(nil, err)` and the handler responds 502.

## Risks / Trade-offs

- [Mirrors that currently redirect from HTTPS to plain HTTP will stop working] → Called out as a separate **Changed** entry in `CHANGELOG.md`. Operators can configure the `http` mirror URL directly if they accept it.
- [An `http` → `http` → `https` → `http` chain is rejected only at the last hop] → Intended: once on HTTPS, the chain must stay on HTTPS.
- [Presigned S3 URLs may expire between hops] → Not a concern in practice: hops happen within milliseconds of each other.
