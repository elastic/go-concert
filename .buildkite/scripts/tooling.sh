#!/bin/bash

retry() {
    local retries=$1
    shift

    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [ $count -lt "$retries" ]; then
            >&2 echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
            sleep $wait
        else
            >&2 echo "Retry $count/$retries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

go_install_method() {
    local version=$1
    version=$(grep -Eo '[0-9]\.[0-9]+\.[0-9]+' <<< "$version")
    minor=$(awk  -F . '{print $2}' <<< "$version")
    major=$(awk  -F . '{print $1}' <<< "$version")
    if [[ $minor -gt 15 && $major -eq 1 || $major -eq 2 ]]; then
        echo "get -u"
    else
        echo "install"
    fi
}
