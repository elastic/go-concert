#!/usr/bin/env bash
set -euxo pipefail

# Prepare enviroment
source .buildkite/scripts/pre-install-command.sh
add_bin_path
with_go_junit_report

# Run the tests
set +e
OUT_FILE="build/test-report.out"
mkdir -p build
go test "./..." -v 2>&1 | tee ${OUT_FILE}
status=$?
set -e

go-junit-report > "build/junit-${SETUP_GOLANG_VERSION}-linux.xml" < ${OUT_FILE}

ls build
exit ${status}
