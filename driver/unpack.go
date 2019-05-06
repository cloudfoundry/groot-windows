package driver

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/archive/tar"

	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) (int64, error) {
	logger.Info("unpack-start")
	defer logger.Info("unpack-finished")

	if d.Store == "" {
		return 0, &EmptyDriverStoreError{}
	}

	di := hcsshim.DriverInfo{HomeDir: d.LayerStore(), Flavour: 1}
	exists, err := d.hcsClient.LayerExists(di, layerID)
	if err != nil {
		return 0, err
	}

	if err := d.privilegeElevator.EnableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}); err != nil {
		return 0, err
	}
	defer d.privilegeElevator.DisableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege})

	outputDir := filepath.Join(d.LayerStore(), layerID)

	if exists {
		logger.Info("layer-id-exists", lager.Data{"layerID": layerID})
		content, err := ioutil.ReadFile(d.layerSizeFile(layerID))
		if err != nil {

			// if the size file does not exist, delete the layer and recreate
			// this way there is an upgrade path from previous groot versions
			if os.IsNotExist(err) {
				logger.Info("removing-out-of-date-layer", lager.Data{"layerID": layerID})
				if err := d.hcsClient.DestroyLayer(di, layerID); err != nil {
					return 0, err
				}
			} else {
				return 0, err
			}
		} else {
			return strconv.ParseInt(string(content), 10, 64)
		}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, err
	}

	parentLayerPaths := []string{}
	for _, id := range parentIDs {
		parentLayerPaths = append([]string{filepath.Join(d.LayerStore(), id)}, parentLayerPaths...)
	}

	layerWriter, err := d.hcsClient.NewLayerWriter(di, layerID, parentLayerPaths)
	if err != nil {
		return 0, err
	}
	defer layerWriter.Close()

	d.tarStreamer.SetReader(layerTar)
	defer d.tarStreamer.SetReader(bytes.NewReader(nil))

	var (
		hdr         *tar.Header
		nextFileErr error
	)

	var totalSize int64
	for {
		if hdr == nil {
			hdr, nextFileErr = d.tarStreamer.Next()
		} else if base := path.Base(hdr.Name); strings.HasPrefix(base, ".wh.") {
			name := filepath.Join(path.Dir(hdr.Name), base[len(".wh."):])
			if err := layerWriter.Remove(name); err != nil {
				return 0, err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else if hdr.Typeflag == tar.TypeLink {
			if err := layerWriter.AddLink(filepath.FromSlash(hdr.Name), filepath.FromSlash(hdr.Linkname)); err != nil {
				return 0, err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else {
			name, size, fileInfo, err := d.tarStreamer.FileInfoFromHeader(hdr)
			if err != nil {
				return 0, err
			}

			if err := layerWriter.Add(filepath.FromSlash(name), fileInfo); err != nil {
				return 0, err
			}

			hdr, nextFileErr = d.tarStreamer.WriteBackupStreamFromTarFile(layerWriter, hdr)
			totalSize += size
		}

		if nextFileErr != nil {
			break
		}
	}

	if nextFileErr != io.EOF {
		return 0, nextFileErr
	}

	err = ioutil.WriteFile(d.layerSizeFile(layerID), []byte(strconv.FormatInt(totalSize, 10)), 0644)
	return totalSize, err
}
