$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

ginkgo.exe -p -r --race -keep-going --randomize-suites --fail-on-pending
$exitCode = $LastExitCode
if ($exitCode -ne 0) {
  Exit $exitCode
}
