package tarstream

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"

	"archive/tar"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/backuptar"
)

var mutatedFiles = map[string]string{
	"UtilityVM/Files/EFI/Microsoft/Boot/BCD":      "bcd.bak",
	"UtilityVM/Files/EFI/Microsoft/Boot/BCD.LOG":  "bcd.log.bak",
	"UtilityVM/Files/EFI/Microsoft/Boot/BCD.LOG1": "bcd.log1.bak",
	"UtilityVM/Files/EFI/Microsoft/Boot/BCD.LOG2": "bcd.log2.bak",
}

type Streamer struct {
	r   *tar.Reader
	buf *bufio.Writer
}

func New() *Streamer {
	return &Streamer{
		r:   tar.NewReader(bytes.NewBuffer(nil)),
		buf: bufio.NewWriter(nil),
	}
}

func (s *Streamer) SetReader(r io.Reader) {
	s.r = tar.NewReader(r)
}

func (s *Streamer) Next() (*tar.Header, error) {
	return s.r.Next()
}

func (s *Streamer) FileInfoFromHeader(hdr *tar.Header) (string, int64, *winio.FileBasicInfo, error) {
	return backuptar.FileInfoFromHeader(hdr)
}

func (s *Streamer) WriteBackupStreamFromTarFile(w io.Writer, hdr *tar.Header, layerPath string) (nextHdr *tar.Header, err error) {
	var backupFile *os.File
	var backupWriter *winio.BackupFileWriter
	if backupPath, ok := mutatedFiles[hdr.Name]; ok {
		backupFile, err = os.Create(filepath.Join(layerPath, backupPath))
		if err != nil {
			return nil, err
		}
		defer func() {
			cerr := backupFile.Close()
			if err == nil {
				err = cerr
			}
		}()
		backupWriter = winio.NewBackupFileWriter(backupFile, false)
		defer func() {
			cerr := backupWriter.Close()
			if err == nil {
				err = cerr
			}
		}()
		s.buf.Reset(io.MultiWriter(w, backupWriter))
	} else {
		s.buf.Reset(w)
	}

	defer func() {
		if ferr := s.buf.Flush(); ferr != nil {
			err = ferr
		}
	}()

	return backuptar.WriteBackupStreamFromTarFile(s.buf, s.r, hdr)
}
