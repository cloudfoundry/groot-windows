package driver

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Stats(logger lager.Logger, bundleID string) (groot.VolumeStats, error) {
	logger.Info("stats-start")
	defer logger.Info("stats-finished")

	di := hcsshim.DriverInfo{HomeDir: d.VolumeStore(), Flavour: 1}
	volumePath, err := d.hcsClient.GetLayerMountPath(di, bundleID)
	if err != nil {
		return groot.VolumeStats{}, err
	} else if volumePath == "" {
		return groot.VolumeStats{}, &MissingVolumePathError{Id: bundleID}
	}

	quotaUsed, err := d.limiter.GetQuotaUsed(volumePath)
	if err != nil {
		return groot.VolumeStats{}, err
	}

	file := filepath.Join(d.VolumeStore(), bundleID, "base_image_size")
	base, err := ioutil.ReadFile(file)
	if err != nil {
		return groot.VolumeStats{}, err
	}

	baseImageSize, err := strconv.ParseInt(string(base), 10, 64)
	if err != nil {
		return groot.VolumeStats{}, fmt.Errorf("couldn't parse base_image_size: %s", err.Error())
	}

	return groot.VolumeStats{DiskUsage: groot.DiskUsage{
		TotalBytesUsed:     baseImageSize + int64(quotaUsed),
		ExclusiveBytesUsed: int64(quotaUsed),
	}}, nil
}
