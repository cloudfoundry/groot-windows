$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

Invoke-Expression "go run github.com/onsi/ginkgo/v2/ginkgo $args"
if ($LastExitCode -ne 0) {
  throw "tests failed"
}
