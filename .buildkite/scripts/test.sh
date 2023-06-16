#!/usr/bin/env bash
set -euxo pipefail

awk --version
sed --version
# Prepare enviroment
source .buildkite/scripts/pre-install-command.sh
add_bin_path
with_go_junit_report

# Run the tests
set +e
export OUT_FILE="build/test-report.out"
mkdir -p build
go test "./..." -v 2>&1 | tee ${OUT_FILE}
status=$?

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > "build/junit-${GO_VERSION}.xml" < ${OUT_FILE}

exit ${status}