## ADDED Requirements

### Requirement: Follow multi-hop upstream redirects
The proxy SHALL follow upstream redirect responses (301, 302, 303, 307, 308) for each mirror attempt, up to a maximum of 10 redirect hops. Every hop SHALL replay the original request method and buffered body, with the same filtered request headers.

#### Scenario: Two-hop redirect chain is followed and cached
- **WHEN** a client requests a cacheable `.rpm` file and the mirror redirects A → B → C, where C returns 200
- **THEN** the proxy returns 200 with C's body to the client
- **AND** the file is written to the cache

#### Scenario: Single redirect still works
- **WHEN** the mirror returns a single redirect to a location that returns 200
- **THEN** the proxy returns 200 with the redirected body

### Requirement: Redirect hop limit
When a mirror is still redirecting after 10 hops, the proxy SHALL treat that mirror attempt as failed and continue with the next mirror.

#### Scenario: Redirect loop exceeds the limit
- **WHEN** the only mirror redirects in a loop that never returns a non-redirect response
- **THEN** the proxy responds with 502 Bad Gateway
- **AND** the client does not receive a 3xx response

#### Scenario: Fallback to next mirror after the limit is exceeded
- **WHEN** mirror 1 exceeds the redirect limit and mirror 2 returns 200
- **THEN** the proxy returns 200 with mirror 2's body

### Requirement: Reject HTTPS to HTTP redirect downgrades
The proxy SHALL reject any redirect hop from an `https` URL to an `http` location, including the first hop, and treat that mirror attempt as failed. A redirect from an `http` URL to another `http` location SHALL still be followed.

#### Scenario: HTTPS mirror redirects to plain HTTP
- **WHEN** the only mirror is served over HTTPS and redirects to an `http` location
- **THEN** the proxy does not request the `http` location
- **AND** the proxy responds with 502 Bad Gateway

#### Scenario: HTTP mirror redirects to HTTP
- **WHEN** a mirror configured with an `http` URL redirects to another `http` location that returns 200
- **THEN** the proxy returns 200 with the redirected body

### Requirement: Never pass redirects to the client
The proxy SHALL NOT forward an upstream 3xx redirect response to the client. If no mirror yields a usable response, the proxy SHALL respond with 502 Bad Gateway.

#### Scenario: All mirrors fail due to redirect errors
- **WHEN** every mirror either exceeds the redirect limit or attempts an HTTPS to HTTP downgrade
- **THEN** the proxy responds with 502 Bad Gateway
