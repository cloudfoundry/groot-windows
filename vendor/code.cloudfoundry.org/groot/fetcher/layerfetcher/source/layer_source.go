package source // import "code.cloudfoundry.org/groot/fetcher/layerfetcher/source"

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"runtime"
	"strings"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
	_ "github.com/containers/image/docker"
	manifestpkg "github.com/containers/image/manifest"
	_ "github.com/containers/image/oci/layout"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	digestpkg "github.com/opencontainers/go-digest"
	imgspec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const MAX_DOCKER_RETRIES = 3

type LayerSource struct {
	skipOCIChecksumValidation bool
	systemContext             types.SystemContext
}

func NewLayerSource(systemContext types.SystemContext, skipOCIChecksumValidation bool) LayerSource {
	return LayerSource{
		systemContext:             systemContext,
		skipOCIChecksumValidation: skipOCIChecksumValidation,
	}
}

func (s *LayerSource) Manifest(logger lager.Logger, imageURL *url.URL) (types.Image, error) {
	logger = logger.Session("fetching-image-manifest", lager.Data{"imageURL": imageURL})
	logger.Info("starting")
	defer logger.Info("ending")

	img, err := s.getImageWithRetries(logger, imageURL)
	if err != nil {
		logger.Error("fetching-image-reference-failed", err)
		return nil, errors.Wrap(err, "fetching image reference")
	}

	img, err = s.convertImage(logger, img, imageURL)
	if err != nil {
		logger.Error("converting-image-failed", err)
		return nil, err
	}

	for i := 0; i < MAX_DOCKER_RETRIES; i++ {
		logger.Debug("attempt-get-config", lager.Data{"attempt": i + 1})
		_, e := img.ConfigBlob()
		if e == nil {
			return img, nil
		}

		logger.Error("fetching-image-config-failed", e, lager.Data{"attempt": i + 1})
		err = e
	}

	return nil, errors.Wrap(err, "fetching image configuration")
}

func (s *LayerSource) Blob(logger lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (string, int64, error) {
	logrus.SetOutput(os.Stderr)
	logger = logger.Session("streaming-blob", lager.Data{
		"imageURL": imageURL,
		"digest":   layerInfo.BlobID,
	})
	logger.Info("starting")
	defer logger.Info("ending")

	imgSrc, err := s.imageSource(logger, imageURL)
	if err != nil {
		return "", 0, err
	}

	blobInfo := types.BlobInfo{
		Digest: digestpkg.Digest(layerInfo.BlobID),
		URLs:   layerInfo.URLs,
	}

	blob, size, err := s.getBlobWithRetries(logger, imgSrc, blobInfo)
	if err != nil {
		return "", 0, err
	}
	defer blob.Close()
	logger.Debug("got-blob-stream", lager.Data{"digest": layerInfo.BlobID, "size": size, "mediaType": layerInfo.MediaType})

	blobTempFile, err := ioutil.TempFile("", "blob-"+layerInfo.BlobID)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		blobTempFile.Close()
		if err != nil {
			os.Remove(blobTempFile.Name())
		}
	}()

	blobIDHash := sha256.New()
	digestReader := ioutil.NopCloser(io.TeeReader(blob, blobIDHash))
	if layerInfo.MediaType == "" || strings.Contains(layerInfo.MediaType, "gzip") {
		logger.Debug("uncompressing-blob")

		digestReader, err = gzip.NewReader(digestReader)
		if err != nil {

			return "", 0, errors.Wrapf(err, "expected blob to be of type %s", layerInfo.MediaType)
		}
		defer digestReader.Close()
	}

	diffIDHash := sha256.New()
	digestReader = ioutil.NopCloser(io.TeeReader(digestReader, diffIDHash))

	if _, err = io.Copy(blobTempFile, digestReader); err != nil {
		logger.Error("writing-blob-to-file", err)
		return "", 0, errors.Wrap(err, "writing blob to tempfile")
	}

	blobIDHex := strings.Split(layerInfo.BlobID, ":")[1]
	if err = s.checkCheckSum(logger, blobIDHash, blobIDHex, imageURL.Scheme); err != nil {
		return "", 0, errors.Wrap(err, "layerID digest mismatch")
	}

	if err = s.checkCheckSum(logger, diffIDHash, layerInfo.DiffID, imageURL.Scheme); err != nil {
		return "", 0, errors.Wrap(err, "diffID digest mismatch")
	}

	return blobTempFile.Name(), size, nil
}

func (s *LayerSource) getBlobWithRetries(logger lager.Logger, imgSrc types.ImageSource, blobInfo types.BlobInfo) (io.ReadCloser, int64, error) {
	var err error
	for i := 0; i < MAX_DOCKER_RETRIES; i++ {
		logger.Debug(fmt.Sprintf("attempt-get-blob-%d", i+1))
		blob, size, e := imgSrc.GetBlob(blobInfo)
		if e == nil {
			logger.Debug("attempt-get-blob-success")
			return blob, size, nil
		}
		err = e
		logger.Error("attempt-get-blob-failed", err)
	}

	return nil, 0, err
}

func (s *LayerSource) checkCheckSum(logger lager.Logger, hash hash.Hash, digest string, scheme string) error {
	if s.skipOCIChecksumValidation && scheme == "oci" {
		return nil
	}

	blobContentsSha := hex.EncodeToString(hash.Sum(nil))
	logger.Debug("checking-checksum", lager.Data{
		"digestIDChecksum":   digest,
		"downloadedChecksum": blobContentsSha,
	})
	if digest != blobContentsSha {
		return errors.Errorf("expected: %s, actual: %s", digest, blobContentsSha)
	}

	return nil
}

func (s *LayerSource) reference(logger lager.Logger, imageURL *url.URL) (types.ImageReference, error) {
	refString := generateRefString(imageURL)
	logger.Debug("parsing-reference", lager.Data{"refString": refString})
	transport := transports.Get(imageURL.Scheme)
	ref, err := transport.ParseReference(refString)
	if err != nil {
		return nil, errors.Wrap(err, "parsing url failed")
	}

	return ref, nil
}

func generateRefString(imageURL *url.URL) string {
	refString := "/"
	if imageURL.Host != "" {
		refString += "/" + imageURL.Host
	}
	refString += imageURL.Path

	if runtime.GOOS == "windows" && imageURL.Scheme == "oci" {
		refString = strings.TrimPrefix(refString, "//")
	}

	return refString
}

func (s *LayerSource) getImageWithRetries(logger lager.Logger, imageURL *url.URL) (types.Image, error) {
	ref, err := s.reference(logger, imageURL)
	if err != nil {
		return nil, err
	}

	var imgErr error
	for i := 0; i < MAX_DOCKER_RETRIES; i++ {
		logger.Debug(fmt.Sprintf("attempt-get-image-%d", i+1))

		img, e := ref.NewImage(&s.systemContext)
		if e == nil {
			logger.Debug("attempt-get-image-success")
			return img, nil
		}
		imgErr = e
	}

	return nil, errors.Wrap(imgErr, "creating image")
}

func (s *LayerSource) imageSource(logger lager.Logger, imageURL *url.URL) (types.ImageSource, error) {
	ref, err := s.reference(logger, imageURL)
	if err != nil {
		return nil, err
	}

	imgSrc, err := ref.NewImageSource(&s.systemContext)
	if err != nil {
		return nil, errors.Wrap(err, "creating image source")
	}

	return imgSrc, nil
}

func (s *LayerSource) convertImage(logger lager.Logger, originalImage types.Image, imageURL *url.URL) (types.Image, error) {
	_, mimetype, err := originalImage.Manifest()
	if err != nil {
		return nil, err
	}

	if mimetype != manifestpkg.DockerV2Schema1MediaType && mimetype != manifestpkg.DockerV2Schema1SignedMediaType {
		return originalImage, nil
	}

	logger = logger.Session("convert-schema-V1-image")
	logger.Info("starting")
	defer logger.Info("ending")

	imgSrc, err := s.imageSource(logger, imageURL)
	if err != nil {
		return nil, err
	}

	diffIDs := []digestpkg.Digest{}
	for _, layer := range originalImage.LayerInfos() {
		diffID, err := s.v1DiffID(logger, layer, imgSrc)
		if err != nil {
			return nil, errors.Wrap(err, "converting V1 schema failed")
		}
		diffIDs = append(diffIDs, diffID)
	}

	options := types.ManifestUpdateOptions{
		ManifestMIMEType: manifestpkg.DockerV2Schema2MediaType,
		InformationOnly: types.ManifestUpdateInformation{
			LayerDiffIDs: diffIDs,
		},
	}

	return originalImage.UpdatedImage(options)
}

func (s *LayerSource) v1DiffID(logger lager.Logger, layer types.BlobInfo, imgSrc types.ImageSource) (digestpkg.Digest, error) {
	blob, _, err := s.getBlobWithRetries(logger, imgSrc, layer)
	if err != nil {
		return "", errors.Wrap(err, "fetching V1 layer blob")
	}
	defer blob.Close()

	gzipReader, err := gzip.NewReader(blob)
	if err != nil {
		return "", errors.Wrap(err, "creating reader for V1 layer blob")
	}

	data, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return "", errors.Wrap(err, "reading V1 layer blob")
	}
	sha := sha256.Sum256(data)

	return digestpkg.NewDigestFromHex("sha256", hex.EncodeToString(sha[:])), nil
}

func preferedMediaTypes() []string {
	return []string{
		imgspec.MediaTypeImageManifest,
		manifestpkg.DockerV2Schema2MediaType,
	}
}
