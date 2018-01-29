package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/windows2016fs/image"
	"code.cloudfoundry.org/windows2016fs/layer"
	metadata "code.cloudfoundry.org/windows2016fs/oci-metadata"
	"code.cloudfoundry.org/windows2016fs/writer"

	"code.cloudfoundry.org/archiver/extractor"

	"github.com/Microsoft/hcsshim"
)

func main() {
	if err := mainBody(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: "+err.Error())
		os.Exit(1)
	}
}

func mainBody() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("Invalid arguments, usage: %s <rootfs-tarball> <output-dir>", os.Args[0])
	}

	rootfstgz := os.Args[1]
	outputDir := os.Args[2]

	imageTempDir, err := ioutil.TempDir("", "hcslayers-oci-image")
	if err != nil {
		return err
	}

	if err := extractor.NewTgz().Extract(rootfstgz, imageTempDir); err != nil {
		return err
	}
	defer os.RemoveAll(imageTempDir)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	mr := metadata.NewReader(imageTempDir)
	lm := layer.NewManager(hcsshim.DriverInfo{HomeDir: outputDir, Flavour: 1}, &writer.Writer{})
	im := image.NewManager(imageTempDir, mr, lm, os.Stderr)

	topLayerId, err := im.Extract()
	if err != nil {
		return err
	}

	fmt.Printf(filepath.Join(outputDir, topLayerId))
	return nil
}
