## ADDED Requirements

### Requirement: Wildcard suffix caches all files
When the `suffixes` list for a repository contains `"*"`, the cache SHALL treat every proxied file as a cache candidate, subject to the `exclude` list.

#### Scenario: File with uncommon extension is cached under wildcard repo
- **WHEN** a request is made for a file with an extension not in any explicit suffix list (e.g. `.crate`) under a repo with `suffixes: ["*"]`
- **THEN** `IsCacheCandidate` returns true

#### Scenario: Wildcard does not affect repos without it
- **WHEN** a request is made for a file under a repo whose `suffixes` list does not contain `"*"`
- **THEN** `IsCacheCandidate` applies the existing suffix-match logic unchanged

### Requirement: Exclude list prevents specific files from being cached
A repository MAY define an `exclude` list. Each entry is matched against the request filename as an exact name first, then as a suffix. If any entry matches, the file SHALL NOT be cached regardless of `suffixes`.

#### Scenario: Exact filename match prevents caching
- **WHEN** a request is made for a file whose name exactly matches an `exclude` entry (e.g. `layout.conf`)
- **THEN** `IsCacheCandidate` returns false

#### Scenario: Suffix match prevents caching
- **WHEN** a request is made for a file whose name ends with an `exclude` entry (e.g. `.sig`)
- **THEN** `IsCacheCandidate` returns false

#### Scenario: Non-matching file is not excluded
- **WHEN** a request is made for a file that does not match any `exclude` entry
- **THEN** the `exclude` list has no effect on the cache candidacy decision

#### Scenario: Exclude applies without wildcard suffix
- **WHEN** a repository has explicit suffixes (no `"*"`) and an `exclude` list, and a request is made for a file that matches both a suffix and an exclude entry
- **THEN** `IsCacheCandidate` returns false (exclude takes precedence)

### Requirement: Explicit suffixes alongside wildcard are redundant but valid
When the `suffixes` list contains both `"*"` and explicit suffix entries, the configuration SHALL be accepted. pkgproxy SHALL log a warning identifying the repository and the redundant entries. Cache behavior is identical to having only `"*"`.

#### Scenario: Mixed wildcard and explicit suffixes triggers a warning
- **WHEN** pkgproxy loads a repository config whose `suffixes` list contains `"*"` and at least one other entry
- **THEN** the repository is accepted without error, a warning is logged naming the repository and the redundant suffixes, and `IsCacheCandidate` behaves as if only `"*"` were present

### Requirement: Exclude field is optional
The `exclude` field in a repository config SHALL be optional. Repositories without it SHALL behave identically to the current behavior.

#### Scenario: Repository without exclude field
- **WHEN** pkgproxy loads a repository config with no `exclude` key
- **THEN** the repository is accepted without error and cache behavior is unchanged
