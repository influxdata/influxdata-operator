#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/tmp/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="influxdata-operator"
REPO_PATH="github.com/dev9-labs/influxdata-operator"
BUILD_PATH="${REPO_PATH}/cmd/${PROJECT_NAME}"
TEST_PATH="${REPO_PATH}/${TEST_LOCATION}"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
if $ENABLE_TESTS ; then
	echo "building "${PROJECT_NAME}-test"..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c -o ${BIN_DIR}/${PROJECT_NAME}-test $TEST_PATH
fi
