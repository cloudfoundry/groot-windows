package driver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager"
)

func (d *Driver) WriteMetadata(logger lager.Logger, bundleID string, volumeData groot.ImageMetadata) error {
	logger.Info("write-metadata-start")
	defer logger.Info("write-metadata-finished")

	if err := d.writeMetadata(d.metadataFile(bundleID), volumeData); err != nil {
		return fmt.Errorf("WriteMetadata failed: %s", err.Error())
	}

	return nil
}

func (d *Driver) writeMetadata(metadataFile string, volumeData groot.ImageMetadata) error {
	data, err := json.Marshal(volumeData)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(metadataFile, data, 0644)
}
