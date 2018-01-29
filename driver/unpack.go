package driver

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/archive/tar"

	"code.cloudfoundry.org/lager"
	"github.com/Microsoft/hcsshim"
)

func (d *Driver) Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) error {
	logger.Info("unpack-start")
	defer logger.Info("unpack-finished")

	outputDir := filepath.Join(d.layerStore, layerID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := d.privilegeElevator.EnableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}); err != nil {
		return err
	}
	defer d.privilegeElevator.DisableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege})

	parentLayerPaths := []string{}
	for _, id := range parentIDs {
		parentLayerPaths = append([]string{filepath.Join(d.layerStore, id)}, parentLayerPaths...)
	}

	di := hcsshim.DriverInfo{HomeDir: d.layerStore, Flavour: 1}
	layerWriter, err := d.hcsClient.NewLayerWriter(di, layerID, parentLayerPaths)
	if err != nil {
		return err
	}
	defer layerWriter.Close()

	d.tarStreamer.SetReader(layerTar)
	defer d.tarStreamer.SetReader(bytes.NewReader(nil))

	var (
		hdr         *tar.Header
		nextFileErr error
	)

	for {
		if hdr == nil {
			hdr, nextFileErr = d.tarStreamer.Next()
		} else if base := path.Base(hdr.Name); strings.HasPrefix(base, ".wh.") {
			name := filepath.Join(path.Dir(hdr.Name), base[len(".wh."):])
			if err := layerWriter.Remove(name); err != nil {
				return err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else if hdr.Typeflag == tar.TypeLink {
			if err := layerWriter.AddLink(filepath.FromSlash(hdr.Name), filepath.FromSlash(hdr.Linkname)); err != nil {
				return err
			}

			hdr, nextFileErr = d.tarStreamer.Next()
		} else {
			name, _, fileInfo, err := d.tarStreamer.FileInfoFromHeader(hdr)
			if err != nil {
				return err
			}

			if err := layerWriter.Add(filepath.FromSlash(name), fileInfo); err != nil {
				return err
			}

			hdr, nextFileErr = d.tarStreamer.WriteBackupStreamFromTarFile(layerWriter, hdr)
		}

		if nextFileErr != nil {
			break
		}
	}

	if nextFileErr != io.EOF {
		return nextFileErr
	}

	return nil
}
