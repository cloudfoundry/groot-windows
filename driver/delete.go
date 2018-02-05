package driver

import (
	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Delete(logger lager.Logger, bundleID string) error {
	logger.Info("delete-start")
	defer logger.Info("delete-finished")

	if d.Store == "" {
		return &EmptyDriverStoreError{}
	}

	di := hcsshim.DriverInfo{HomeDir: d.VolumeStore(), Flavour: 1}
	exists, err := d.hcsClient.LayerExists(di, bundleID)
	if err != nil {
		return err
	}

	if !exists {
		logger.Info("volume-not-found", lager.Data{"bundleID": bundleID})
		return nil
	}

	return d.hcsClient.DestroyLayer(di, bundleID)
}
