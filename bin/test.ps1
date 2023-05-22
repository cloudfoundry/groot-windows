$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

ginkgo.exe -p -r --race -keep-going --randomize-suites --fail-on-pending
if ($LastExitCode -ne 0) {
  throw "tests failed"
}
