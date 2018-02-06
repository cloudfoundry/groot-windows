package filefetcher // import "code.cloudfoundry.org/groot/fetcher/filefetcher"

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
)

type FileFetcher struct {
}

func NewFileFetcher() *FileFetcher {
	return &FileFetcher{}
}

func (l *FileFetcher) StreamBlob(logger lager.Logger, imageURL *url.URL,
	layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
	logger = logger.Session("stream-blob")
	logger.Info("starting", lager.Data{
		"imageURL": imageURL.String(),
		"source":   layerInfo.BlobID,
	})
	defer logger.Info("ending")

	imagePath := imageURL.String()
	if _, err := os.Stat(imagePath); err != nil {
		return nil, 0, errors.Wrapf(err, "local image not found in `%s`", imagePath)
	}

	if err := l.validateImage(imagePath); err != nil {
		return nil, 0, errors.Wrap(err, "invalid base image")
	}

	logger.Debug("opening-tar", lager.Data{"imagePath": imagePath})
	stream, err := os.Open(imagePath)
	if err != nil {
		return nil, 0, errors.Wrap(err, "reading local image")
	}

	return stream, 0, nil
}

func (l *FileFetcher) ImageInfo(logger lager.Logger, imageURL *url.URL) (imagepuller.ImageInfo, error) {
	logger = logger.Session("layers-digest", lager.Data{"imageURL": imageURL.String()})
	logger.Info("starting")
	defer logger.Info("ending")

	stat, err := os.Stat(imageURL.String())
	if err != nil {
		return imagepuller.ImageInfo{},
			errors.Wrap(err, "fetching image timestamp")
	}

	return imagepuller.ImageInfo{
		LayerInfos: []imagepuller.LayerInfo{
			imagepuller.LayerInfo{
				BlobID:        imageURL.String(),
				ParentChainID: "",
				ChainID:       l.generateChainID(imageURL.String(), stat.ModTime().UnixNano()),
				Size:          stat.Size(),
			},
		},
	}, nil
}

func (l *FileFetcher) generateChainID(imagePath string, timestamp int64) string {
	imagePathSha := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", imagePath, timestamp)))
	return hex.EncodeToString(imagePathSha[:])
}

func (l *FileFetcher) validateImage(imagePath string) error {
	stat, err := os.Stat(imagePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return errors.New("directory provided instead of a tar file")
	}

	return nil
}
