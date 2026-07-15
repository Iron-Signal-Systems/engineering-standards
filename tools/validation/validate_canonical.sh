#!/usr/bin/env bash
set -Eeuo pipefail

printf '%s\n' \
  'Canonical validation has not been configured for this repository.' \
  'Define the exact native host or VM profile and replace this bootstrap refusal.' >&2
exit 2
