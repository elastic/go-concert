#!/bin/bash
set -euo pipefail

source .buildkite/scripts/tooling.sh

add_bin_path(){
    mkdir -p "${WORKSPACE}/bin"
    export PATH="${WORKSPACE}/bin:${PATH}"
}

with_go_junit_report() {
    go env
    echo $PATH
    export PATH="${PATH}:${GOPATH}:${GOPATH}/bin"
    version=$(go version)
    install_method=$(go_install_method "$version")
    go ${install_method} github.com/jstemmer/go-junit-report
}

WORKSPACE=${WORKSPACE:-"$(pwd)"}
