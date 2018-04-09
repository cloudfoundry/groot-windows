package ondemand

import "io"

type Reader struct {
	Create func() (io.ReadCloser, error)

	reader io.ReadCloser
}

func (d *Reader) Read(p []byte) (int, error) {
	if d.reader == nil {
		reader, err := d.Create()
		if err != nil {
			return 0, err
		}

		d.reader = reader
	}

	return d.reader.Read(p)
}

func (d *Reader) Close() error {
	if d.reader == nil {
		return nil
	}
	return d.reader.Close()
}
