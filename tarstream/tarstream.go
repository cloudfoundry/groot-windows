package tarstream

import (
	"bufio"
	"bytes"
	"io"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/archive/tar"
	"github.com/Microsoft/go-winio/backuptar"
)

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

func (s *Streamer) WriteBackupStreamFromTarFile(w io.Writer, hdr *tar.Header) (*tar.Header, error) {
	s.buf.Reset(w)
	hdr, err := backuptar.WriteBackupStreamFromTarFile(s.buf, s.r, hdr)
	if err != nil {
		return nil, err
	}

	if err := s.buf.Flush(); err != nil {
		return nil, err
	}
	return hdr, nil
}
