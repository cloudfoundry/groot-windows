package driver

import (
	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Exists(logger lager.Logger, layerID string) bool {
	logger.Info("exists-start")
	defer logger.Info("exists-finished")

	di := hcsshim.DriverInfo{HomeDir: d.LayerStore(), Flavour: 1}
	exists, err := d.hcsClient.LayerExists(di, layerID)
	if err != nil {
		logger.Error("error-checking-layer", err)
		return false
	}

	if exists {
		logger.Info("layer-id-exists", lager.Data{"layerID": layerID})
	}

	return exists
}
