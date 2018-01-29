package registry

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	tokenURL    = "%s/token?service=registry.docker.io&scope=repository:%s:pull"
	manifestURL = "%s/v2/%s/manifests/%s"
	blobURL     = "%s/v2/%s/blobs/%s"
)

const (
	foreignLayer   = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"
	diffLayer      = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	imageConfig    = "application/vnd.docker.container.image.v1+json"
	manifestV2     = "application/vnd.docker.distribution.manifest.v2+json"
	manifestV2List = "application/vnd.docker.distribution.manifest.list.v2+json"
)

type Registry struct {
	authServerURL     string
	registryServerURL string
	imageName         string
	imageTag          string
}

func New(authServerURL, registryServerURL, imageName, imageTag string) *Registry {
	return &Registry{
		authServerURL:     authServerURL,
		registryServerURL: registryServerURL,
		imageName:         imageName,
		imageTag:          imageTag,
	}
}

func (r *Registry) Manifest() (v1.Manifest, error) {
	var m v1.Manifest
	buffer := new(bytes.Buffer)

	if err := r.downloadResource(r.manifestURL(), buffer, manifestV2, manifestV2List); err != nil {
		return v1.Manifest{}, err
	}

	if err := json.Unmarshal(buffer.Bytes(), &m); err != nil {
		return v1.Manifest{}, err
	}

	return m, nil
}

func (r *Registry) Config(config v1.Descriptor) (v1.Image, error) {
	configSHA, err := getLayerSHA(config.Digest)
	if err != nil {
		return v1.Image{}, &DownloadError{Cause: err, blobSHA: configSHA}
	}

	if config.MediaType != imageConfig {
		return v1.Image{}, &DownloadError{Cause: &InvalidMediaTypeError{mediaType: config.MediaType}, blobSHA: configSHA}
	}

	buffer := new(bytes.Buffer)

	if err := r.downloadResource(r.blobURL(config.Digest), buffer); err != nil {
		return v1.Image{}, &DownloadError{Cause: err, blobSHA: configSHA}
	}

	recievedSHA := fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes()))
	if configSHA != recievedSHA {
		return v1.Image{}, &DownloadError{Cause: &SHAMismatchError{expected: configSHA, actual: recievedSHA}, blobSHA: configSHA}
	}

	var i v1.Image
	if err := json.Unmarshal(buffer.Bytes(), &i); err != nil {
		return v1.Image{}, err
	}

	return i, nil
}

func (r *Registry) DownloadLayer(layer v1.Descriptor, outputDir string) error {
	layerSHA, err := getLayerSHA(layer.Digest)
	if err != nil {
		return &DownloadError{Cause: err, blobSHA: layerSHA}
	}

	layerFile := filepath.Join(outputDir, layerSHA)
	if err := r.downloadLayer(layer, layerFile); err != nil {
		return &DownloadError{Cause: err, blobSHA: layerSHA}
	}

	if err := checkSHA256(layerFile, layerSHA); err != nil {
		return &DownloadError{Cause: err, blobSHA: layerSHA}
	}
	return nil
}

func (r *Registry) downloadLayer(layer v1.Descriptor, outputFile string) error {
	var layerURL string

	switch layer.MediaType {
	case diffLayer:
		layerURL = r.blobURL(layer.Digest)
	case foreignLayer:
		layerURL = layer.URLs[0]
	default:
		return &InvalidMediaTypeError{mediaType: layer.MediaType}
	}

	f, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := r.downloadResource(layerURL, f); err != nil {
		return err
	}
	return nil
}

func (r *Registry) manifestURL() string {
	return fmt.Sprintf(manifestURL, r.registryServerURL, r.imageName, r.imageTag)
}

func (r *Registry) blobURL(d digest.Digest) string {
	return fmt.Sprintf(blobURL, r.registryServerURL, r.imageName, d)
}

func (r *Registry) downloadResource(url string, output io.Writer, acceptMediaTypes ...string) error {
	token, err := r.getToken()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+token)
	for _, mediaType := range acceptMediaTypes {
		req.Header.Add("Accept", mediaType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HTTPNotOKError{statusCode: resp.StatusCode}
	}
	_, err = io.Copy(output, resp.Body)

	return err
}

func (r *Registry) getToken() (string, error) {
	resp, err := http.Get(fmt.Sprintf(tokenURL, r.authServerURL, r.imageName))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &HTTPNotOKError{statusCode: resp.StatusCode}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var token struct {
		Token string
	}

	if err := json.Unmarshal(body, &token); err != nil {
		return "", err
	}

	return token.Token, nil
}

func checkSHA256(file, expected string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	sum := fmt.Sprintf("%x", h.Sum(nil))
	if sum != expected {
		return &SHAMismatchError{expected: expected, actual: sum}
	}
	return nil
}

func getLayerSHA(d digest.Digest) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}

	if d.Algorithm() != digest.SHA256 {
		return "", &DigestAlgorithmError{expected: digest.SHA256, actual: d.Algorithm()}
	}
	return d.Encoded(), nil
}
