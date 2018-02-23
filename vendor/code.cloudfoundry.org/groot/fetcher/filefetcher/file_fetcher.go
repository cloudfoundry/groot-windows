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
	imagePath string
}

func NewFileFetcher(imageURL *url.URL) *FileFetcher {
	return &FileFetcher{imagePath: imageURL.String()}
}

func (l *FileFetcher) StreamBlob(logger lager.Logger, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
	logger = logger.Session("stream-blob", lager.Data{"imagePath": l.imagePath})
	logger.Info("starting", lager.Data{
		"source": layerInfo.BlobID,
	})
	defer logger.Info("ending")

	if _, err := os.Stat(l.imagePath); err != nil {
		return nil, 0, errors.Wrapf(err, "local image not found in `%s`", l.imagePath)
	}

	if err := l.validateImage(); err != nil {
		return nil, 0, errors.Wrap(err, "invalid base image")
	}

	logger.Debug("opening-tar", lager.Data{"imagePath": l.imagePath})
	stream, err := os.Open(l.imagePath)
	if err != nil {
		return nil, 0, errors.Wrap(err, "reading local image")
	}

	return stream, 0, nil
}

func (l *FileFetcher) ImageInfo(logger lager.Logger) (imagepuller.ImageInfo, error) {
	logger = logger.Session("layers-digest", lager.Data{"imagePath": l.imagePath})

	logger.Info("starting")
	defer logger.Info("ending")

	stat, err := os.Stat(l.imagePath)
	if err != nil {
		return imagepuller.ImageInfo{},
			errors.Wrap(err, "fetching image timestamp")
	}

	return imagepuller.ImageInfo{
		LayerInfos: []imagepuller.LayerInfo{
			imagepuller.LayerInfo{
				BlobID:        l.imagePath,
				ParentChainID: "",
				ChainID:       l.generateChainID(stat.ModTime().UnixNano()),
				Size:          stat.Size(),
			},
		},
	}, nil
}

func (l *FileFetcher) Close() error {
	return nil
}

func (l *FileFetcher) generateChainID(timestamp int64) string {
	imagePathSha := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", l.imagePath, timestamp)))
	return hex.EncodeToString(imagePathSha[:])
}

func (l *FileFetcher) validateImage() error {
	stat, err := os.Stat(l.imagePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return errors.New("directory provided instead of a tar file")
	}

	return nil
}
