# ACM Policy OpenShift Tests Extension (OTE)

This directory contains the OpenShift Tests Extension (OTE) binary for ACM Policy testing.

## Overview

The `acm-policy-tests` binary implements the OTE v1.1 protocol so ACM Policy checks can run under `openshift-tests` with `--extension-local-binaries`.

## Structure

```
openshift-tests-ext/
├── cmd/acm-policy-tests/   # Binary entry point
├── pkg/adapter/            # Generic test execution adapter
├── test/acm/               # Test scenarios (JSON)
├── scripts/                # Test scripts
├── go.mod                  # Go module definition
└── Makefile                # Build automation
```

## Building

```bash
make build
```

## Running Tests

### Standalone (for development)

```bash
./acm-policy-tests run-suite acm/all
./acm-policy-tests run-suite acm/smoke
./acm-policy-tests list tests
./acm-policy-tests info
```

### With openshift-tests (production)

```bash
openshift-tests run-suite acm/all \
  --extension-local-binaries /usr/bin/acm-policy-tests.gz \
  --junit-dir ./junit
```

## Test Suites

- `acm/all` - All ACM Policy tests
- `acm/smoke` - ACM Policy smoke tests (subset with SMOKE tag)

## Adding Tests

Tests are defined in `test/acm/scenarios.json`. To add a new test:

1. Create a test script in `scripts/`
2. Add a scenario entry to `scenarios.json` using `${INTEROP_SCRIPTS_DIR}/your-script.sh`
3. Rebuild the binary with `make build`

Scripts are embedded in the binary. The adapter extracts them to a temp directory at runtime and sets `INTEROP_SCRIPTS_DIR`, so the path is correct when the binary is invoked via `--extension-local-binaries`. Shell scripts use exit `2` for skip conditions (for example missing `KUBECONFIG` or ACM not installed); the adapter maps that to OTE `skipped`.

## CI Integration

This binary is built as part of CI composite containers and invoked by step-registry refs in openshift/release:

- `ci-operator/step-registry/interop-tests/opp-acm--policy--all`
- `ci-operator/step-registry/interop-tests/opp-acm--policy--smoke`

## Related

- JIRA: https://redhat.atlassian.net/browse/INTEROP-9588
- OpenShift Tests Extension framework: https://github.com/openshift-eng/openshift-tests-extension
- OTE Protocol spec: https://github.com/openshift-eng/openshift-tests-extension/blob/main/docs/protocol.md
