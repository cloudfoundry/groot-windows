package driver

import (
	"code.cloudfoundry.org/lager"
)

func (d *Driver) Exists(logger lager.Logger, layerID string) bool {
	return false
}
