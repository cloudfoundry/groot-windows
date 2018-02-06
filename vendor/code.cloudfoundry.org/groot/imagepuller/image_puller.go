package imagepuller // import "code.cloudfoundry.org/groot/imagepuller"

import (
	"io"
	"net/url"

	"code.cloudfoundry.org/lager"
	imgspec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

//go:generate counterfeiter . Fetcher
//go:generate counterfeiter . VolumeDriver

type LayerInfo struct {
	BlobID        string
	ChainID       string
	DiffID        string
	ParentChainID string
	Size          int64
	URLs          []string
	MediaType     string
}

type ImageInfo struct {
	LayerInfos []LayerInfo
	Config     imgspec.Image
}

type VolumeMeta struct {
	Size int64
}

type Fetcher interface {
	ImageInfo(logger lager.Logger, imageURL *url.URL) (ImageInfo, error)
	StreamBlob(logger lager.Logger, imageURL *url.URL, layerInfo LayerInfo) (io.ReadCloser, int64, error)
}

type VolumeDriver interface {
	Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) error
	Exists(logger lager.Logger, layerID string) bool
}

type Image struct {
	Image         imgspec.Image
	ChainIDs      []string
	BaseImageSize int64
}

type ImageSpec struct {
	DiskLimit             int64
	ExcludeImageFromQuota bool
	ImageSrc              *url.URL
}

type ImagePuller struct {
	fetcher      Fetcher
	volumeDriver VolumeDriver
}

func NewImagePuller(fetcher Fetcher, volumeDriver VolumeDriver) *ImagePuller {
	return &ImagePuller{
		fetcher:      fetcher,
		volumeDriver: volumeDriver,
	}
}

func (p *ImagePuller) Pull(logger lager.Logger, spec ImageSpec) (Image, error) {
	logger = logger.Session("image-pulling", lager.Data{"spec": spec})
	logger.Info("starting")
	defer logger.Info("ending")

	imageInfo, err := p.fetcher.ImageInfo(logger, spec.ImageSrc)
	if err != nil {
		return Image{}, errors.Wrap(err, "fetching list of layer infos")
	}
	logger.Debug("fetched-layer-infos", lager.Data{"infos": imageInfo.LayerInfos})

	if err = p.quotaExceeded(logger, imageInfo.LayerInfos, spec); err != nil {
		return Image{}, err
	}

	err = p.buildLayer(logger, len(imageInfo.LayerInfos)-1, imageInfo.LayerInfos, spec)
	if err != nil {
		return Image{}, err
	}
	chainIDs := p.chainIDs(imageInfo.LayerInfos)

	image := Image{
		Image:         imageInfo.Config,
		ChainIDs:      chainIDs,
		BaseImageSize: p.layersSize(imageInfo.LayerInfos),
	}
	return image, nil
}

func (p *ImagePuller) quotaExceeded(logger lager.Logger, layerInfos []LayerInfo, spec ImageSpec) error {
	if spec.ExcludeImageFromQuota || spec.DiskLimit == 0 {
		return nil
	}

	totalSize := p.layersSize(layerInfos)
	if totalSize > spec.DiskLimit {
		err := errors.Errorf("layers exceed disk quota %d/%d bytes", totalSize, spec.DiskLimit)
		logger.Error("blob-manifest-size-check-failed", err, lager.Data{
			"totalSize":             totalSize,
			"diskLimit":             spec.DiskLimit,
			"excludeImageFromQuota": spec.ExcludeImageFromQuota,
		})
		return err
	}

	return nil
}

func (p *ImagePuller) chainIDs(layerInfos []LayerInfo) []string {
	chainIDs := []string{}
	for _, layerInfo := range layerInfos {
		chainIDs = append(chainIDs, layerInfo.ChainID)
	}
	return chainIDs
}

func (p *ImagePuller) buildLayer(logger lager.Logger, index int, layerInfos []LayerInfo, spec ImageSpec) error {
	if index < 0 {
		return nil
	}

	layerInfo := layerInfos[index]
	logger = logger.Session("build-layer")
	logger.Debug("buildlayer-started", lager.Data{
		"blobID":        layerInfo.BlobID,
		"chainID":       layerInfo.ChainID,
		"parentChainID": layerInfo.ParentChainID,
	})

	if p.volumeDriver.Exists(logger, layerInfo.ChainID) {
		return nil
	}

	downloadChan := make(chan downloadReturn, 1)
	go p.downloadLayer(logger, spec, layerInfo, downloadChan)

	// buildLayerErr is deliberately checked later so we can cleanup properly
	buildLayerErr := p.buildLayer(logger, index-1, layerInfos, spec)

	downloadResult := <-downloadChan
	if downloadResult.Err != nil {
		return downloadResult.Err
	}
	defer downloadResult.Stream.Close()

	if buildLayerErr != nil {
		return buildLayerErr
	}

	parentChainIDs := []string{}
	if index != 0 {
		for i := 0; i < index; i++ {
			parentChainIDs = append(parentChainIDs, layerInfos[i].ChainID)
		}
	}

	return p.volumeDriver.Unpack(logger, layerInfos[index].ChainID, parentChainIDs, downloadResult.Stream)
}

type downloadReturn struct {
	Stream io.ReadCloser
	Err    error
}

func (p *ImagePuller) downloadLayer(logger lager.Logger, spec ImageSpec, layerInfo LayerInfo, downloadChan chan downloadReturn) {
	logger = logger.Session("downloading-layer", lager.Data{"LayerInfo": layerInfo})
	logger.Debug("starting")
	defer logger.Debug("ending")

	stream, size, err := p.fetcher.StreamBlob(logger, spec.ImageSrc, layerInfo)
	if err != nil {
		err = errors.Wrapf(err, "streaming blob `%s`", layerInfo.BlobID)
	}

	logger.Debug("got-stream-for-blob", lager.Data{
		"size":                  size,
		"diskLimit":             spec.DiskLimit,
		"excludeImageFromQuota": spec.ExcludeImageFromQuota,
	})

	downloadChan <- downloadReturn{Stream: stream, Err: err}
}

func (p *ImagePuller) layersSize(layerInfos []LayerInfo) int64 {
	var totalSize int64
	for _, layerInfo := range layerInfos {
		totalSize += layerInfo.Size
	}
	return totalSize
}
