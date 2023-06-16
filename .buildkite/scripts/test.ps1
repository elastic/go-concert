$ErrorActionPreference = "Stop" # set -e
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf input
    git rm --quiet --cached -r .
    git reset --quiet --hard
}

function withGolang($version) {
    Write-Host "-- Install golang --"
    choco install -y golang --version $version
    $env:ChocolateyInstall = Convert-Path "$((Get-Command choco).Path)\..\.."
    Import-Module "$env:ChocolateyInstall\helpers\chocolateyProfile.psm1"
    refreshenv
    go version
    go env
}

function goInstallMethod($version) {
    $regexp = '\d+\.\d+.\d+'
    $match = [regex]::Match($version, $regexp)

    if ($match.Success) {
        Write-Host $match.Value
        $split_version = $match.Value -split '\.'
        $major = $split_version[0]
        $minor = $split_version[1]
        if ($minor -gt 15 -and $major -eq 1 -or $major -eq 2) {
            return "get -u"
        }
        return "install"
    }
}

function withGoJUnitReport {
    Write-Host "-- Install go-junit-report --"
    $version = go version
    $method = goInstallMethod $version
    echo $method
    if ($method = "install") {
        go install github.com/jstemmer/go-junit-report/v2
    } else {
        go get -u github.com/jstemmer/go-junit-report/v2
    }
}

# Prepare enviroment
fixCRLF
withGolang $env:SETUP_GOLANG_VERSION
withGoJUnitReport

# Run test
$ErrorActionPreference = "Continue" # set +e
mkdir -p build
$OUT_FILE="build\output-report.out"
go test "./..." -v > $OUT_FILE | type $OUT_FILE
$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Get-Content $OUT_FILE | go-junit-report > "build\uni-junit-$env:SETUP_GOLANG_VERSION.xml"
Get-Content "build\uni-junit-$env:SETUP_GOLANG_VERSION.xml" -Encoding Unicode | Set-Content -Encoding UTF8 "build\junit-$env:SETUP_GOLANG_VERSION-win.xml"
Remove-Item "build\uni-junit-$env:SETUP_GOLANG_VERSION.xml", "$OUT_FILE"
