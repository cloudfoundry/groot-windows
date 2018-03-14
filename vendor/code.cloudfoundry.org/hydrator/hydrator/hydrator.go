package hydrator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/hydrator/compress"
	"code.cloudfoundry.org/hydrator/downloader"
	metadata "code.cloudfoundry.org/hydrator/oci-metadata"
	"code.cloudfoundry.org/hydrator/registry"
)

type Hydrator struct {
	outDir    string
	imageName string
	imageTag  string
	noTarball bool
}

func New(outDir, imageName, imageTag string, noTarball bool) *Hydrator {
	return &Hydrator{
		outDir:    outDir,
		imageName: imageName,
		imageTag:  imageTag,
		noTarball: noTarball,
	}
}

func (h *Hydrator) Run() error {
	if err := os.MkdirAll(h.outDir, 0755); err != nil {
		return errors.New("Could not create output directory")
	}

	if h.imageName == "" {
		return errors.New("No image name provided")
	}

	var imageDownloadDir string
	if h.noTarball {
		imageDownloadDir = h.outDir
	} else {
		tempDir, err := ioutil.TempDir("", "hydrate")
		if err != nil {
			return fmt.Errorf("Could not create tmp dir: %s", tempDir)
		}
		defer os.RemoveAll(tempDir)

		imageDownloadDir = tempDir
	}

	blobDownloadDir := filepath.Join(imageDownloadDir, "blobs", "sha256")
	if err := os.MkdirAll(blobDownloadDir, 0755); err != nil {
		return err
	}

	r := registry.New("https://auth.docker.io", "https://registry.hub.docker.com", h.imageName, h.imageTag)
	d := downloader.New(blobDownloadDir, r, os.Stdout)

	layers, diffIds, err := d.Run()
	if err != nil {
		return err
	}

	w := metadata.NewWriter(imageDownloadDir, layers, diffIds)
	if err := w.Write(); err != nil {
		return err
	}
	fmt.Printf("\nAll layers downloaded.\n")

	if !h.noTarball {
		nameParts := strings.Split(h.imageName, "/")
		if len(nameParts) != 2 {
			return errors.New("Invalid image name")
		}
		outFile := filepath.Join(h.outDir, fmt.Sprintf("%s-%s.tgz", nameParts[1], h.imageTag))

		fmt.Printf("Writing %s...\n", outFile)

		c := compress.New()
		if err := c.WriteTgz(imageDownloadDir, outFile); err != nil {
			return err
		}

		fmt.Println("Done.")
	}

	return nil
}
