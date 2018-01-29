package writer

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/archive/tar"
	"github.com/Microsoft/go-winio/backuptar"
	"github.com/Microsoft/hcsshim"
)

type Writer struct {
	layerWriter hcsshim.LayerWriter
}

func (w *Writer) WriteLayer(layerGzipFile string) error {
	if err := winio.EnableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}); err != nil {
		return err
	}
	defer winio.DisableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege})

	gf, err := os.Open(layerGzipFile)
	if err != nil {
		return err
	}
	defer gf.Close()

	gr, err := gzip.NewReader(gf)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	hdr, err := tr.Next()
	buf := bufio.NewWriter(nil)

	for err == nil {
		base := path.Base(hdr.Name)
		if strings.HasPrefix(base, ".wh.") {
			name := path.Join(path.Dir(hdr.Name), base[len(".wh."):])
			err = w.layerWriter.Remove(filepath.FromSlash(name))
			if err != nil {
				return fmt.Errorf("Failed to remove: %s", err.Error())
			}
			hdr, err = tr.Next()
		} else if hdr.Typeflag == tar.TypeLink {
			err = w.layerWriter.AddLink(filepath.FromSlash(hdr.Name), filepath.FromSlash(hdr.Linkname))
			if err != nil {
				return fmt.Errorf("Failed to add link: %s", err.Error())
			}
			hdr, err = tr.Next()
		} else {
			var (
				name     string
				fileInfo *winio.FileBasicInfo
			)
			name, _, fileInfo, err = backuptar.FileInfoFromHeader(hdr)
			if err != nil {
				return fmt.Errorf("Failed to get file info: %s", err.Error())
			}
			err = w.layerWriter.Add(filepath.FromSlash(name), fileInfo)
			if err != nil {
				return fmt.Errorf("Failed to add layer: %s", err.Error())
			}
			buf.Reset(w.layerWriter)

			hdr, err = backuptar.WriteBackupStreamFromTarFile(buf, tr, hdr)
			ferr := buf.Flush()
			if ferr != nil {
				err = ferr
			}
		}
	}

	if err != io.EOF {
		return err
	}

	return w.layerWriter.Close()
}

func (w *Writer) SetHCSLayerWriter(di hcsshim.DriverInfo, layerId string, parentLayerPaths []string) error {
	if err := winio.EnableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}); err != nil {
		return err
	}
	defer winio.DisableProcessPrivileges([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege})

	hcsWriter, err := hcsshim.NewLayerWriter(di, layerId, parentLayerPaths)
	if err != nil {
		return err
	}

	w.layerWriter = hcsWriter
	return nil
}
