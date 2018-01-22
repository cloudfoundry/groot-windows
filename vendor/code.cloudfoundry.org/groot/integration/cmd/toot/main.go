package main

import (
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/toot/toot"
)

func main() {
	driver := &toot.Toot{BaseDir: os.Getenv("TOOT_BASE_DIR")}
	groot.Run(driver, os.Args)
}
