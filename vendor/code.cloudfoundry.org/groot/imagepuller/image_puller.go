package imagepuller // import "code.cloudfoundry.org/groot/imagepuller"

import (
	"io"

	"code.cloudfoundry.org/groot/imagepuller/ondemand"
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
	ImageInfo(logger lager.Logger) (ImageInfo, error)
	StreamBlob(logger lager.Logger, layerInfo LayerInfo) (io.ReadCloser, int64, error)
	Close() error
}

type VolumeDriver interface {
	Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) (int64, error)
}

type Image struct {
	Config   imgspec.Image
	ChainIDs []string
	Size     int64
}

type ImageSpec struct {
	DiskLimit             int64
	ExcludeImageFromQuota bool
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

	imageInfo, err := p.fetcher.ImageInfo(logger)
	if err != nil {
		return Image{}, errors.Wrap(err, "fetching list of layer infos")
	}
	logger.Debug("fetched-layer-infos", lager.Data{"infos": imageInfo.LayerInfos})

	if err = quotaExceeded(logger, imageInfo.LayerInfos, spec); err != nil {
		return Image{}, err
	}

	imageSize, err := p.buildLayers(logger, imageInfo.LayerInfos, spec)
	if err != nil {
		return Image{}, err
	}
	chainIDs := chainIDs(imageInfo.LayerInfos)

	image := Image{
		Config:   imageInfo.Config,
		ChainIDs: chainIDs,
		Size:     imageSize,
	}
	return image, nil
}

func (p *ImagePuller) buildLayers(logger lager.Logger, layerInfos []LayerInfo, spec ImageSpec) (int64, error) {
	totalBytes := int64(0)

	for i, layerInfo := range layerInfos {
		builtBytes, err := p.buildLayer(logger, layerInfo, chainIDs(layerInfos[0:i]), spec)
		if err != nil {
			return 0, err
		}
		totalBytes += builtBytes
	}

	return totalBytes, nil
}

func (p *ImagePuller) buildLayer(logger lager.Logger, layerInfo LayerInfo, parentChainIDs []string, spec ImageSpec) (int64, error) {
	logger = logger.Session("build-layer", lager.Data{
		"blobID":        layerInfo.BlobID,
		"chainID":       layerInfo.ChainID,
		"parentChainID": layerInfo.ParentChainID,
	})

	onDemandReader := &ondemand.Reader{
		Create: func() (io.ReadCloser, error) {
			stream, blobSize, err := p.fetcher.StreamBlob(logger, layerInfo)
			if err != nil {
				return nil, errors.Wrapf(err, "opening stream for blob `%s`", layerInfo.BlobID)
			}

			logger.Debug("got-stream-for-blob", lager.Data{"size": blobSize})
			return stream, nil
		},
	}
	defer onDemandReader.Close()

	return p.volumeDriver.Unpack(logger, layerInfo.ChainID, parentChainIDs, onDemandReader)
}

func chainIDs(layerInfos []LayerInfo) []string {
	chainIDs := []string{}
	for _, layerInfo := range layerInfos {
		chainIDs = append(chainIDs, layerInfo.ChainID)
	}
	return chainIDs
}

func quotaExceeded(logger lager.Logger, layerInfos []LayerInfo, spec ImageSpec) error {
	if spec.ExcludeImageFromQuota || spec.DiskLimit == 0 {
		return nil
	}

	totalSize := layersSize(layerInfos)
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

func layersSize(layerInfos []LayerInfo) int64 {
	var totalSize int64
	for _, layerInfo := range layerInfos {
		totalSize += layerInfo.Size
	}
	return totalSize
}
