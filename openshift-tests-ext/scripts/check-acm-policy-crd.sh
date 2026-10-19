#!/bin/bash
set -euxo pipefail
shopt -s inherit_errexit 2>/dev/null || true

if [[ -z "${KUBECONFIG:-}" ]]; then
    echo "KUBECONFIG not set, skipping" >&2
    exit 2
fi

if ! oc get crd policies.policy.open-cluster-management.io &>/dev/null; then
    echo "FAIL: CRD policies.policy.open-cluster-management.io is not installed" >&2
    exit 1
fi

echo "PASS: ACM Policy CRD is installed"
exit 0
