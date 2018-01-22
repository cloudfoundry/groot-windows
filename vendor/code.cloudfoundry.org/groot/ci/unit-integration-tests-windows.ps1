$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

$env:GOPATH = $PWD
$env:PATH = $env:GOPATH + "/bin;" + $env:PATH

go version

push-location "src\code.cloudfoundry.org\groot"
  # We don't pass -u, so this won't fetch a later revision of groot than the one
  # we are supposed to be testing.
  go get -v -t
  if ($LastExitCode -ne 0) {
    exit $LastExitCode
  }
pop-location

go get github.com/golang/protobuf
Write-Host "Installing Ginkgo"
go install ./src/github.com/onsi/ginkgo/ginkgo
if ($LastExitCode -ne 0) {
    throw "Ginkgo installation process returned error code: $LastExitCode"
}

./src/code.cloudfoundry.org/groot/scripts/test.ps1 # -race
Exit $LastExitCode
