## Requirements

### Requirement: GitHub Actions e2e workflow with workflow_dispatch trigger
The project SHALL provide a GitHub Actions workflow at `.github/workflows/e2e.yaml` that runs e2e tests. The workflow SHALL be triggered only via `workflow_dispatch` (manual trigger). The workflow SHALL NOT run automatically on push or pull request events.

#### Scenario: Workflow is triggered manually
- **WHEN** a user triggers the e2e workflow via the GitHub Actions UI or API
- **THEN** the workflow starts and runs the e2e test matrix

#### Scenario: Workflow does not run on push
- **WHEN** code is pushed to any branch
- **THEN** the e2e workflow is NOT triggered

### Requirement: Matrix strategy with one job per distro/release tuple
The workflow SHALL use a GitHub Actions matrix strategy with one job per distro/release tuple. Each matrix entry SHALL define a display name, the Go test function name to run, and the release version. Each job SHALL appear as a separate entry in the GitHub Actions UI for clear failure isolation.

#### Scenario: Matrix produces separate jobs
- **WHEN** the e2e workflow is triggered
- **THEN** GitHub Actions creates one job per matrix entry, each visible as a separate row in the workflow run UI

#### Scenario: Each job runs exactly one distro test
- **WHEN** a matrix job executes
- **THEN** it runs only the Go test function specified by the matrix entry (e.g., `TestFedora`) with the release specified by `E2E_RELEASE`

### Requirement: 5-minute timeout per matrix job
Each matrix job SHALL have a `timeout-minutes` of 5.

#### Scenario: Job exceeds timeout
- **WHEN** a matrix job runs longer than 5 minutes
- **THEN** the job is cancelled by GitHub Actions

### Requirement: Matrix includes all supported distro/release tuples
The workflow matrix SHALL include the following distro/release tuples: Fedora 43, CentOS Stream 10, AlmaLinux 10, Rocky Linux 10, Debian trixie, Ubuntu noble, Arch Linux latest.

#### Scenario: All distros are represented in the matrix
- **WHEN** the e2e workflow is triggered
- **THEN** jobs are created for Fedora 43, CentOS Stream 10, AlmaLinux 10, Rocky Linux 10, Debian trixie, Ubuntu noble, and Arch Linux latest

### Requirement: Workflow job steps
Each matrix job SHALL checkout the repository, set up Go, and run the e2e test for the specific distro using `go test -tags e2e -run <TestFunction> ./test/e2e/` with the `E2E_RELEASE` environment variable set from the matrix.

#### Scenario: Job executes test with correct parameters
- **WHEN** a matrix job for Fedora 43 runs
- **THEN** it executes `go test -tags e2e -run TestFedora ./test/e2e/` with `E2E_RELEASE=43`
