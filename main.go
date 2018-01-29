package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/hcs"
	"code.cloudfoundry.org/groot-windows/privilege"
	"code.cloudfoundry.org/groot-windows/tarstream"
)

func main() {
	driver := driver.New(os.Getenv("GROOT_BASE_DIR"), &hcs.Client{}, tarstream.New(), &privilege.Elevator{})
	groot.Run(driver, os.Args)
}
