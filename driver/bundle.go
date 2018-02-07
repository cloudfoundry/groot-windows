package driver

import (
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (d *Driver) Bundle(logger lager.Logger, bundleID string, layerIDs []string, bundleSpec groot.BundleSpec) (specs.Spec, error) {
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

	if err := d.hcsClient.CreateLayer(di, bundleID, layerFolders[0], layerFolders); err != nil {
		return specs.Spec{}, err
	}

	volumePath, err := d.hcsClient.GetLayerMountPath(di, bundleID)
	if err != nil {
		return specs.Spec{}, err
	} else if volumePath == "" {
		return specs.Spec{}, &MissingVolumePathError{Id: bundleID}
	}

	if err := d.setQuota(volumePath, bundleSpec); err != nil {
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

func (d *Driver) setQuota(volumePath string, bundleSpec groot.BundleSpec) error {
	if bundleSpec.DiskLimit == 0 {
		return nil
	}

	if bundleSpec.DiskLimit < 0 {
		return &InvalidDiskLimitError{Limit: bundleSpec.DiskLimit}
	}

	quota := uint64(bundleSpec.DiskLimit)
	if !bundleSpec.ExcludeImageFromQuota {
		if bundleSpec.DiskLimit <= bundleSpec.BaseImageSize {
			return &DiskLimitTooSmallError{Limit: bundleSpec.DiskLimit, Base: bundleSpec.BaseImageSize}
		}
		quota = quota - uint64(bundleSpec.BaseImageSize)
	}

	return d.limiter.SetQuota(volumePath, quota)
}
