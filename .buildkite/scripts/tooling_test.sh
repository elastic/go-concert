#!/bin/bash
set -euo pipefail

source .buildkite/scripts/tooling.sh

pass=true

method=$(go_install_method "1.15.8")
if [[ ${method} != "install" ]]; then
    echo "Expected method for 1.15.8 'install'. Got: $method"
    pass=false
fi

method=$(go_install_method "1.20.0")
if [[ ${method} != "get -u" ]]; then
    echo "Expected method for 1.20.0 'get -u'. Got: $method"
    pass=false
fi

method=$(go_install_method "2.5.0")
if [[ ${method} != "get -u" ]]; then
    echo "Expected method for 2.5.0 'get -u'. Got: $method"
    pass=false
fi

method=$(go_install_method "go version go1.20.4 darwin/arm64")
if [[ ${method} != "get -u" ]]; then
    echo "Expected method for 1.20.4 'get -u'. Got: $method"
    pass=false
fi

if [[ $pass == "false" ]]; then
    echo "Got test errors ^"
    exit 1
fi

echo "Test PASS"