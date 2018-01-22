package fetcher

import (
	"io"
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
)

type Fetcher struct {
	FileFetcher  imagepuller.Fetcher
	LayerFetcher imagepuller.Fetcher
}

func (f *Fetcher) ImageInfo(logger lager.Logger, imageURL *url.URL) (imagepuller.ImageInfo, error) {
	return f.fetcher(imageURL).ImageInfo(logger, imageURL)
}

func (f *Fetcher) StreamBlob(logger lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
	return f.fetcher(imageURL).StreamBlob(logger, imageURL, layerInfo)
}

func (f *Fetcher) fetcher(imageURL *url.URL) imagepuller.Fetcher {
	if imageURL.Scheme == "oci" || imageURL.Scheme == "docker" {
		return f.LayerFetcher
	}

	return f.FileFetcher
}
