package compress

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Compressor struct{}

func New() *Compressor {
	return &Compressor{}
}

func (c *Compressor) WriteTgz(srcDir, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return writeDirToTar(srcDir, tw, "")
}

func writeDirToTar(srcDir string, dest *tar.Writer, prefix string) error {
	dir, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, fi := range files {
		source := filepath.Join(dir, fi.Name())
		hdr := tarHeader(fi, prefix)

		if err := addTarFile(source, &hdr, dest); err != nil {
			return err
		}

		if fi.IsDir() {
			if err := writeDirToTar(source, dest, filepath.Join(prefix, fi.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

const c_ISREG = 0100000 // Regular file
const c_ISDIR = 040000  // Directory

func tarHeader(fi os.FileInfo, prefix string) tar.Header {
	filename := filepath.Join(prefix, fi.Name())

	// use linux style path separators so tar headers are always identical
	// tar on windows will handle these paths correctly
	linuxFilename := strings.Replace(filename, "\\", "/", -1)

	if fi.IsDir() {
		return tar.Header{
			Name:     linuxFilename + "/",
			Mode:     0755 | c_ISDIR,
			Typeflag: tar.TypeDir,
		}
	}

	return tar.Header{
		Name:     linuxFilename,
		Mode:     0644 | c_ISREG,
		Size:     fi.Size(),
		Typeflag: tar.TypeReg,
	}
}

func addTarFile(file string, hdr *tar.Header, tw *tar.Writer) error {
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if hdr.Typeflag != tar.TypeReg {
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(tw, f)
	if err != nil {
		return err
	}

	return nil
}
