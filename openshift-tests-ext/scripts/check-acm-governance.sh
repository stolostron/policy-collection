#!/bin/bash
set -euxo pipefail
shopt -s inherit_errexit 2>/dev/null || true

if [[ -z "${KUBECONFIG:-}" ]]; then
    echo "KUBECONFIG not set, skipping" >&2
    exit 2
fi

csvName=$(oc get csv --all-namespaces --no-headers 2>/dev/null | grep -E "governance-policy|advanced-cluster-management|multiclusterhub-operator" | head -1 | awk '{print $2}')
if [[ -z "${csvName}" ]]; then
    echo "WARN: ACM governance / MultiClusterHub CSV not found — ACM Policy may not be deployed" >&2
    exit 2
fi

csvPhase=$(oc get csv --all-namespaces -o jsonpath="{.items[?(@.metadata.name==\"${csvName}\")].status.phase}" 2>/dev/null | awk '{print $1}')
if [[ "${csvPhase}" != "Succeeded" ]]; then
    echo "FAIL: ACM governance CSV is not Succeeded (phase: ${csvPhase})" >&2
    exit 1
fi

echo "PASS: ACM governance policy framework is installed (${csvName})"
exit 0
