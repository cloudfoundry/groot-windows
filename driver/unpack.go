package driver

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"archive/tar"

	winio "github.com/Microsoft/go-winio"

	"code.cloudfoundry.org/lager/v3"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) (int64, error) {
	logger.Info("unpack-start")
	defer logger.Info("unpack-finished")

	if d.Store == "" {
		return 0, &EmptyDriverStoreError{}
	}

	logger.Debug("building-driver-info")
	di := hcsshim.DriverInfo{HomeDir: d.LayerStore(), Flavour: 1}
	logger.Debug("checking-if-layer-exists")
	exists, err := d.hcsClient.LayerExists(di, layerID)
	if err != nil {
		return 0, err
	}

	logger.Debug("elevating-privileges")
	if err := d.privilegeElevator.EnableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}); err != nil {
		return 0, err
	}
	defer func() {
		logger.Debug("disabling-process-privileges")
		err := d.privilegeElevator.DisableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege})
		if err != nil {
			logger.Error("error-disabling-process-privileges", err, lager.Data{"privs": []string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}})
		}
		logger.Debug("Disabled-process-privileges")
	}()

	outputDir := filepath.Join(d.LayerStore(), layerID)

	if exists {
		logger.Info("layer-id-exists", lager.Data{"layerID": layerID})
		content, err := os.ReadFile(d.layerSizeFile(layerID))
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
	defer func() {
		logger.Debug("closing-layer-writer")
		err := layerWriter.Close()
		if err != nil {
			logger.Error("error-closing-layer-writer", err)
		}
		logger.Debug("closed-layer-writer")
	}()

	d.tarStreamer.SetReader(layerTar)
	defer func() {
		logger.Debug("unsetting-tarstreamer-reader")
		d.tarStreamer.SetReader(bytes.NewReader(nil))
		logger.Debug("set-tarstreamer-reader-to-nil")
	}()

	var (
		hdr         *tar.Header
		nextFileErr error
	)

	var totalSize int64
	logger.Debug("entering-tar-for-loop")
	for {
		if hdr == nil {
			logger.Debug("hdr-is-nil")
			hdr, nextFileErr = d.tarStreamer.Next()
		} else if base := path.Base(hdr.Name); strings.HasPrefix(base, ".wh.") {
			logger.Debug("base-has-prefix-.wh.")
			name := filepath.Join(path.Dir(hdr.Name), base[len(".wh."):])
			if err := layerWriter.Remove(name); err != nil {
				return 0, err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else if hdr.Typeflag == tar.TypeLink {
			logger.Debug("hdr.Type-is-link")
			if err := layerWriter.AddLink(filepath.FromSlash(hdr.Name), filepath.FromSlash(hdr.Linkname)); err != nil {
				return 0, err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else {
			logger.Debug("hdr-catch-all")
			name, size, fileInfo, err := d.tarStreamer.FileInfoFromHeader(hdr)
			if err != nil {
				return 0, err
			}

			logger.Debug("adding-path-to-layer", lager.Data{"name": name})
			if err := layerWriter.Add(filepath.FromSlash(name), fileInfo); err != nil {
				return 0, err
			}

			logger.Debug("write-backup-stream-from-tar")
			hdr, nextFileErr = d.tarStreamer.WriteBackupStreamFromTarFile(layerWriter, hdr)
			totalSize += size
		}

		if nextFileErr != nil {
			break
		}
	}
	logger.Debug("out-of-tar-for-loop")

	if nextFileErr != io.EOF {
		return 0, nextFileErr
	}

	logger.Debug("writing-out-layer-file", lager.Data{"file": d.layerSizeFile(layerID), "size": strconv.FormatInt(totalSize, 10)})
	err = os.WriteFile(d.layerSizeFile(layerID), []byte(strconv.FormatInt(totalSize, 10)), 0644)
	logger.Debug("wrote-layer-file")
	return totalSize, err
}
