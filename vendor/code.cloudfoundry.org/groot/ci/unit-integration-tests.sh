#!/bin/bash
set -euo pipefail

export GOPATH=$PWD

go version

# We don't pass -u, so this won't fetch a later revision of groot than the one
# we are supposed to be testing.
go get -v -t code.cloudfoundry.org/groot/...
./src/code.cloudfoundry.org/groot/scripts/test -race
