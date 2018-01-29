package hydrator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/windows2016fs/compress"
	"code.cloudfoundry.org/windows2016fs/downloader"
	metadata "code.cloudfoundry.org/windows2016fs/oci-metadata"
	"code.cloudfoundry.org/windows2016fs/registry"
)

type Hydrator struct {
	outDir    string
	imageName string
	imageTag  string
}

func New(outDir, imageName, imageTag string) *Hydrator {
	return &Hydrator{
		outDir:    outDir,
		imageName: imageName,
		imageTag:  imageTag,
	}
}

func (h *Hydrator) Run() error {
	if err := os.MkdirAll(h.outDir, 0755); err != nil {
		return errors.New("Could not create output directory")
	}

	if h.imageName == "" {
		return errors.New("No image name provided")
	}

	nameParts := strings.Split(h.imageName, "/")
	if len(nameParts) != 2 {
		return errors.New("Invalid image name")
	}
	outFile := filepath.Join(h.outDir, fmt.Sprintf("%s-%s.tgz", nameParts[1], h.imageTag))

	tempDir, err := ioutil.TempDir("", "hydrate")
	if err != nil {
		return fmt.Errorf("Could not create tmp dir: %s", tempDir)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "blobs", "sha256")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return err
	}

	r := registry.New("https://auth.docker.io", "https://registry.hub.docker.com", h.imageName, h.imageTag)
	d := downloader.New(downloadDir, r, os.Stdout)

	layers, diffIds, err := d.Run()
	if err != nil {
		return err
	}

	w := metadata.NewWriter(tempDir, layers, diffIds)
	if err := w.Write(); err != nil {
		return err
	}

	fmt.Printf("\nAll layers downloaded, writing %s...\n", outFile)

	c := compress.New()
	if err := c.WriteTgz(tempDir, outFile); err != nil {
		return err
	}

	fmt.Println("Done.")
	return nil
}
