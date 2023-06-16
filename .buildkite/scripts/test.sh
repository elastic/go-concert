#!/usr/bin/env bash
set -euxo pipefail

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

install_method=$(go_install_method "$SETUP_GOLANG_VERSION")
go ${install_method} github.com/jstemmer/go-junit-report
go-junit-report > "build/junit-${SETUP_GOLANG_VERSION}.xml" < ${OUT_FILE}

exit ${status}