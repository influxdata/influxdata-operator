#!/bin/bash -e

HACK_DIR=$(dirname "${BASH_SOURCE[0]}")
REPO_ROOT="${HACK_DIR}/.."

"${REPO_ROOT}/vendor/k8s.io/code-generator/generate-groups.sh" \
  all \
  github.com/dev9/prod/influxdata-operator/pkg/generated \
  github.com/dev9/prod/influxdata-operator/pkg/apis \
  influxdata:v1alpha1 \
  "$@"