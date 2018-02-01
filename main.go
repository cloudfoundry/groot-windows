package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
)

func main() {
	creator := &driver.Creator{}
	groot.Run(creator, os.Args)
}
