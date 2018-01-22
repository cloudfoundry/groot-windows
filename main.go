package main

import (
	"os"

	"code.cloudfoundry.org/groot-windows/plugin"

	"code.cloudfoundry.org/groot"
)

func main() {
	driver := &plugin.Plugin{}
	groot.Run(driver, os.Args)
}
