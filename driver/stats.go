package driver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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

	data, err := ioutil.ReadFile(d.metadataFile(bundleID))
	if err != nil {
		return groot.VolumeStats{}, err
	}

	var volumeData groot.ImageMetadata
	if err := json.Unmarshal(data, &volumeData); err != nil {
		return groot.VolumeStats{}, fmt.Errorf("couldn't parse metadata.json: %s", err.Error())
	}

	return groot.VolumeStats{DiskUsage: groot.DiskUsage{
		TotalBytesUsed:     volumeData.Size + int64(quotaUsed),
		ExclusiveBytesUsed: int64(quotaUsed),
	}}, nil
}
