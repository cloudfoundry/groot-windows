$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

cd "$psscriptroot\.."
ginkgo -r -keepGoing -failOnPending -randomizeAllSpecs -randomizeSuites -p # "$@"
