package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/hcs"
	"code.cloudfoundry.org/groot-windows/privilege"
	"code.cloudfoundry.org/groot-windows/tarstream"
	"code.cloudfoundry.org/groot-windows/volume"
	"github.com/urfave/cli"
)

func main() {
	driver := driver.New(hcs.NewClient(), tarstream.New(), &privilege.Elevator{}, &volume.Limiter{})

	driverFlags := []cli.Flag{
		cli.StringFlag{
			Name:        "driver-store",
			Value:       "",
			Usage:       "driver store path",
			Destination: &driver.Store,
		}}
	groot.Run(driver, os.Args, driverFlags)
}
