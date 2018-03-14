package downloader

import (
	"fmt"
	"io"
	"sync"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

//go:generate counterfeiter . Registry
type Registry interface {
	Manifest() (v1.Manifest, error)
	Config(v1.Descriptor) (v1.Image, error)
	DownloadLayer(v1.Descriptor, string) error
}

type Downloader struct {
	downloadDir string
	registry    Registry
	logger      io.Writer
}

func New(downloadDir string, registry Registry, logger io.Writer) *Downloader {
	d := &Downloader{
		downloadDir: downloadDir,
		registry:    registry,
		logger:      logger,
	}
	return d
}

func (d *Downloader) Run() ([]v1.Descriptor, []digest.Digest, error) {
	registryManifest, err := d.registry.Manifest()
	if err != nil {
		return nil, nil, err
	}

	registryConfig, err := d.registry.Config(registryManifest.Config)
	if err != nil {
		return nil, nil, err
	}

	if registryConfig.OS != "windows" {
		return nil, nil, fmt.Errorf("invalid container OS: %s", registryConfig.OS)
	}
	if registryConfig.Architecture != "amd64" {
		return nil, nil, fmt.Errorf("invalid container arch: %s", registryConfig.Architecture)
	}

	totalLayers := len(registryManifest.Layers)
	diffIds := registryConfig.RootFS.DiffIDs

	if totalLayers != len(diffIds) {
		return nil, nil, fmt.Errorf("mismatch: %d layers, %d diffIds", totalLayers, len(diffIds))
	}

	fmt.Fprintf(d.logger, "Downloading %d layers...\n", totalLayers)
	wg := sync.WaitGroup{}
	errChan := make(chan error, 1)

	downloadedLayers := []v1.Descriptor{}

	for i, layer := range registryManifest.Layers {
		l := layer
		diffId := diffIds[i]

		ociLayer := v1.Descriptor{
			MediaType: v1.MediaTypeImageLayerGzip,
			Size:      l.Size,
			Digest:    l.Digest,
		}

		downloadedLayers = append(downloadedLayers, ociLayer)

		wg.Add(1)
		go func() {
			fmt.Fprintf(d.logger, "Layer diffID: %.8s, sha256: %.8s begin\n", diffId.Encoded(), l.Digest.Encoded())
			defer wg.Done()
			if err := d.registry.DownloadLayer(l, d.downloadDir); err != nil {
				errChan <- err
				return
			}
			fmt.Fprintf(d.logger, "Layer diffID: %.8s, sha256: %.8s end\n", diffId.Encoded(), l.Digest.Encoded())
		}()
	}

	wgEmpty := make(chan interface{}, 1)
	go func() {
		wg.Wait()
		wgEmpty <- nil
	}()

	select {
	case <-wgEmpty:
	case downloadErr := <-errChan:
		return nil, nil, downloadErr
	}

	return downloadedLayers, diffIds, nil
}
