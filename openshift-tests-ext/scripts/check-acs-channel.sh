#!/bin/bash
set -euo pipefail
shopt -s inherit_errexit 2>/dev/null || true

# INTEROP-9655: ACS channel pin single-channel assertion
# Canonical suite: lp-interop--OPP--policy-collection-acs-channel per ADR 0007 (no doubling)
# Asserts that the rendered ACS OperatorPolicy contains exactly one channel key
# and that it is pinned to the expected value (stable for RHACS 5.0.0 per 2026-11-12 set).
# Uses yq when available, falls back to grep -c; fails the step if count != 1.

SUITE="lp-interop--OPP--policy-collection-acs-channel"
EXPECTED_CHANNEL="stable"
ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp}"
JUNIT_FILE="${ARTIFACT_DIR}/junit_${SUITE}.xml"
# Fallback only when ARTIFACT_DIR is explicitly set to current dir; do not pollute repo on default /tmp runs
JUNIT_FILE_FALLBACK=""
if [[ "${ARTIFACT_DIR}" == "." ]]; then
  JUNIT_FILE_FALLBACK="./junit_${SUITE}.xml"
fi

# Locate source file relative to repo root or script location
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
SOURCE_FILE="${REPO_ROOT}/policygenerator/policy-sets/stable/openshift-plus/input-acs-central/policy-acs-operator-central.yaml"

# Prefer rendered manifests via kustomize if available, otherwise source file
RENDERED_FILE=""
if command -v kustomize &>/dev/null; then
  RENDERED_FILE="$(mktemp)"
  # shellcheck disable=SC2064
  trap "rm -f '${RENDERED_FILE}'" EXIT
  POLICYSET_DIR="${REPO_ROOT}/policygenerator/policy-sets/stable/openshift-plus"
  if kustomize build --enable-alpha-plugins "${POLICYSET_DIR}" >"${RENDERED_FILE}" 2>/dev/null; then
    echo "Rendered manifests via kustomize to ${RENDERED_FILE}"
  else
    echo "WARN: kustomize build failed, falling back to source file" >&2
    RENDERED_FILE=""
  fi
fi

if [[ -n "${RENDERED_FILE}" && -s "${RENDERED_FILE}" ]]; then
  TARGET_FILE="${RENDERED_FILE}"
  echo "Checking rendered manifests: ${TARGET_FILE}"
else
  TARGET_FILE="${SOURCE_FILE}"
  echo "Checking source file: ${TARGET_FILE}"
fi

if [[ ! -f "${TARGET_FILE}" ]]; then
  echo "FAIL: target file not found: ${TARGET_FILE}" >&2
  exit 1
fi

# Count channel occurrences via grep -c (primary assertion per task)
# Must be exactly 1; duplicate channel from insert-after-name sed would yield 2
# For rendered multi-doc manifests, scope to the ACS OperatorPolicy (operatorpolicy-rhacs-operator)
COUNT_FILE="${TARGET_FILE}"
IS_RENDERED="false"
if [[ -n "${RENDERED_FILE:-}" && "${TARGET_FILE}" == "${RENDERED_FILE}" ]]; then
  IS_RENDERED="true"
fi
if [[ "${IS_RENDERED}" == "true" && $(grep -c "channel:" "${TARGET_FILE}" || true) -gt 1 ]]; then
  # Rendered file contains multiple policies (ACS + compliance etc.); isolate ACS OperatorPolicy
  ACS_TMP="$(mktemp)"
  if command -v yq &>/dev/null; then
    yq eval 'select(.metadata.name == "operatorpolicy-rhacs-operator")' "${TARGET_FILE}" >"${ACS_TMP}" 2>/dev/null || true
    if [[ -s "${ACS_TMP}" ]]; then
      COUNT_FILE="${ACS_TMP}"
      echo "Scoped to ACS OperatorPolicy via yq: ${COUNT_FILE}"
    else
      rm -f "${ACS_TMP}" || true
    fi
  fi
  if [[ "${COUNT_FILE}" == "${TARGET_FILE}" ]]; then
    # Fallback: grep -A isolate (25 lines after the ACS operator name)
    ACS_TMP2="$(mktemp)"
    grep -A 30 "name: operatorpolicy-rhacs-operator" "${TARGET_FILE}" | head -n 30 >"${ACS_TMP2}" 2>/dev/null || true
    if grep -q "channel:" "${ACS_TMP2}" 2>/dev/null; then
      COUNT_FILE="${ACS_TMP2}"
      echo "Scoped to ACS OperatorPolicy via grep: ${COUNT_FILE}"
    else
      rm -f "${ACS_TMP2}" || true
    fi
  fi
fi
CHANNEL_COUNT_GREP="$(grep -c "channel:" "${COUNT_FILE}" || true)"
echo "channel count (grep -c): ${CHANNEL_COUNT_GREP} (file: ${COUNT_FILE})"

# yq single-channel assertion when yq is available
CHANNEL_COUNT_YQ=""
CHANNEL_VALUE_YQ=""
if command -v yq &>/dev/null; then
  # yq v4 syntax: eval all documents, count subscription.channel
  CHANNEL_VALUE_YQ="$(yq eval 'select(.spec.subscription.channel != null) | .spec.subscription.channel' "${COUNT_FILE}" 2>/dev/null | head -n 1 || true)"
  CHANNEL_COUNT_YQ="$(yq eval 'select(.spec.subscription.channel != null) | .spec.subscription.channel' "${COUNT_FILE}" 2>/dev/null | grep -c . || true)"
  echo "channel value (yq): ${CHANNEL_VALUE_YQ:-<empty>}"
  echo "channel count (yq): ${CHANNEL_COUNT_YQ}"
  if [[ -n "${CHANNEL_COUNT_YQ}" && "${CHANNEL_COUNT_YQ}" != "${CHANNEL_COUNT_GREP}" ]]; then
    echo "WARN: yq count (${CHANNEL_COUNT_YQ}) differs from grep count (${CHANNEL_COUNT_GREP})" >&2
  fi
fi

# Determine pass/fail
FAIL_REASON=""
if [[ "${CHANNEL_COUNT_GREP}" != "1" ]]; then
  FAIL_REASON="expected exactly 1 channel entry, got ${CHANNEL_COUNT_GREP} (duplicate channel key indicates insert-after-name sed bug)"
elif [[ -n "${CHANNEL_VALUE_YQ}" && "${CHANNEL_VALUE_YQ}" != "${EXPECTED_CHANNEL}" ]]; then
  FAIL_REASON="channel value '${CHANNEL_VALUE_YQ}' != expected '${EXPECTED_CHANNEL}'"
else
  # Also verify grep line contains expected channel when yq not available
  if [[ -z "${CHANNEL_VALUE_YQ}" ]]; then
    if ! grep -q "channel: ${EXPECTED_CHANNEL}" "${TARGET_FILE}"; then
      FAIL_REASON="channel: ${EXPECTED_CHANNEL} not found in ${TARGET_FILE}"
    fi
  fi
fi

# Emit JUnit with canonical suite name without doubling
# The suite name is exactly lp-interop--OPP--policy-collection-acs-channel;
# do not set LP_IO__ET_PPP__NEW_TS_NAME to prefix it again (ExitTrap would double to lp-interop--OPP--lp-interop--OPP--...)
mkdir -p "$(dirname "${JUNIT_FILE}")"
TESTCASE_NAME="acs-channel-pin-is-single-stable"
if [[ -z "${FAIL_REASON}" ]]; then
  echo "PASS: ACS channel pin is single and stable"
  cat >"${JUNIT_FILE}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="${SUITE}" tests="1" failures="0" errors="0" skipped="0">
  <testcase classname="${SUITE}" name="${TESTCASE_NAME}" time="0">
    <system-out>channel count=1 value=${EXPECTED_CHANNEL} file=${TARGET_FILE}</system-out>
  </testcase>
</testsuite>
EOF
  if [[ -n "${JUNIT_FILE_FALLBACK}" && "${JUNIT_FILE}" != "${JUNIT_FILE_FALLBACK}" ]]; then
    cp "${JUNIT_FILE}" "${JUNIT_FILE_FALLBACK}" 2>/dev/null || true
  fi
  echo "JUnit written to ${JUNIT_FILE}"
  exit 0
else
  echo "FAIL: ${FAIL_REASON}" >&2
  cat >"${JUNIT_FILE}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="${SUITE}" tests="1" failures="1" errors="0" skipped="0">
  <testcase classname="${SUITE}" name="${TESTCASE_NAME}" time="0">
    <failure message="${FAIL_REASON}">channel count=${CHANNEL_COUNT_GREP} expected=1 value=${CHANNEL_VALUE_YQ:-unknown} file=${TARGET_FILE}
grep output:
$(grep "channel:" "${TARGET_FILE}" || true)
</failure>
  </testcase>
</testsuite>
EOF
  if [[ -n "${JUNIT_FILE_FALLBACK}" && "${JUNIT_FILE}" != "${JUNIT_FILE_FALLBACK}" ]]; then
    cp "${JUNIT_FILE}" "${JUNIT_FILE_FALLBACK}" 2>/dev/null || true
  fi
  echo "JUnit (failure) written to ${JUNIT_FILE}" >&2
  exit 1
fi
