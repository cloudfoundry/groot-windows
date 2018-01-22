package layerfetcher // import "code.cloudfoundry.org/groot/fetcher/layerfetcher"

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strings"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"

	"github.com/containers/image/types"
	imgspec "github.com/opencontainers/image-spec/specs-go/v1"
	errorspkg "github.com/pkg/errors"
)

const cfBaseDirectoryAnnotation = "org.cloudfoundry.experimental.image.base-directory"

//go:generate counterfeiter . Source
//go:generate counterfeiter . Manifest

type Manifest interface {
	// Manifest is just a shortcut for the types.Image interface,
	// to make it simpler to test with fakes.
	types.Image
}

type Source interface {
	Manifest(logger lager.Logger, imageURL *url.URL) (types.Image, error)
	Blob(logger lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (string, int64, error)
}

type LayerFetcher struct {
	source Source
}

func NewLayerFetcher(source Source) *LayerFetcher {
	return &LayerFetcher{
		source: source,
	}
}

func (f *LayerFetcher) ImageInfo(logger lager.Logger, imageURL *url.URL) (imagepuller.ImageInfo, error) {
	logger = logger.Session("layers-digest", lager.Data{"imageURL": imageURL})
	logger.Info("starting")
	defer logger.Info("ending")

	logger.Debug("fetching-image-manifest")
	manifest, err := f.source.Manifest(logger, imageURL)
	if err != nil {
		return imagepuller.ImageInfo{}, err
	}

	logger.Debug("fetching-image-config")
	var config *imgspec.Image
	config, err = manifest.OCIConfig()
	if err != nil {
		return imagepuller.ImageInfo{}, err
	}

	return imagepuller.ImageInfo{
		LayerInfos: f.createLayerInfos(logger, manifest, config),
		Config:     *config,
	}, nil
}

func (f *LayerFetcher) StreamBlob(logger lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
	logger = logger.Session("streaming", lager.Data{"imageURL": imageURL})
	logger.Info("starting")
	defer logger.Info("ending")

	blobFilePath, size, err := f.source.Blob(logger, imageURL, layerInfo)
	if err != nil {
		logger.Error("source-blob-failed", err, lager.Data{"imageUrl": imageURL, "blobId": layerInfo.BlobID, "URL": layerInfo.URLs})
		return nil, 0, err
	}

	blobReader, err := NewBlobReader(blobFilePath)
	if err != nil {
		logger.Error("blob-reader-failed", err)
		return nil, 0, errorspkg.Wrap(err, "opening stream from temporary blob file")
	}

	return blobReader, size, nil
}

func (f *LayerFetcher) createLayerInfos(logger lager.Logger, image Manifest, config *imgspec.Image) []imagepuller.LayerInfo {
	layerInfos := []imagepuller.LayerInfo{}

	var parentChainID string
	for i, layer := range image.LayerInfos() {
		if i == 0 {
			parentChainID = ""
		}

		diffID := config.RootFS.DiffIDs[i]
		chainID := f.chainID(diffID.String(), parentChainID)
		layerInfos = append(layerInfos, imagepuller.LayerInfo{
			BlobID:        layer.Digest.String(),
			Size:          layer.Size,
			ChainID:       chainID,
			DiffID:        diffID.Hex(),
			ParentChainID: parentChainID,
			URLs:          layer.URLs,
			MediaType:     layer.MediaType,
		})
		parentChainID = chainID
	}

	return layerInfos
}

func (f *LayerFetcher) chainID(diffID string, parentChainID string) string {
	if diffID != "" {
		diffID = strings.Split(diffID, ":")[1]
	}
	chainID := diffID

	if parentChainID != "" {
		chainIDSha := sha256.Sum256([]byte(fmt.Sprintf("%s %s", parentChainID, diffID)))
		chainID = hex.EncodeToString(chainIDSha[:32])
	}

	return chainID
}
