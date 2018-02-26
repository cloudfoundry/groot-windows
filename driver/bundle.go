package driver

import (
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (d *Driver) Bundle(logger lager.Logger, bundleID string, layerIDs []string, diskLimit int64) (specs.Spec, error) {
	logger.Info("bundle-start")
	defer logger.Info("bundle-finished")

	if d.Store == "" {
		return specs.Spec{}, &EmptyDriverStoreError{}
	}
	if err := os.MkdirAll(d.VolumeStore(), 0755); err != nil {
		return specs.Spec{}, err
	}
	di := hcsshim.DriverInfo{HomeDir: d.VolumeStore(), Flavour: 1}

	exists, err := d.hcsClient.LayerExists(di, bundleID)
	if err != nil {
		return specs.Spec{}, err
	}
	if exists {
		return specs.Spec{}, &LayerExistsError{Id: bundleID}
	}

	layerFolders := []string{}
	for _, layerID := range layerIDs {
		layerFolders = append([]string{filepath.Join(d.LayerStore(), layerID)}, layerFolders...)
	}

	cleanupLayer := func() {
		destroyErr := d.hcsClient.DestroyLayer(di, bundleID)
		if destroyErr != nil {
			logger.Error("destroy-failed", destroyErr)
		}
	}

	if err := d.hcsClient.CreateLayer(di, bundleID, layerFolders); err != nil {
		cleanupLayer()
		return specs.Spec{}, err
	}

	volumePath, err := d.hcsClient.GetLayerMountPath(di, bundleID)
	if err != nil {
		cleanupLayer()
		return specs.Spec{}, err
	} else if volumePath == "" {
		cleanupLayer()
		return specs.Spec{}, &MissingVolumePathError{Id: bundleID}
	}

	if err := d.limiter.SetQuota(volumePath, uint64(diskLimit)); err != nil {
		cleanupLayer()
		return specs.Spec{}, err
	}

	return specs.Spec{
		Version: specs.Version,
		Root: &specs.Root{
			Path: volumePath,
		},
		Windows: &specs.Windows{
			LayerFolders: layerFolders,
		},
	}, nil
}
