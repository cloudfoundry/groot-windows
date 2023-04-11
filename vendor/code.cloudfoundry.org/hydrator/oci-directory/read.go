package directory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

func (h *Handler) ReadMetadata() (oci.Manifest, oci.Image, error) {
	i, err := h.loadIndex()
	if err != nil {
		return oci.Manifest{}, oci.Image{}, fmt.Errorf("couldn't load index.json: %s", err.Error())
	}

	mDesc := i.Manifests[0]
	m, err := h.loadManifest(mDesc)
	if err != nil {
		return oci.Manifest{}, oci.Image{}, fmt.Errorf("couldn't load manifest: %s", err.Error())
	}

	c, err := h.loadConfig(m.Config)
	if err != nil {
		return oci.Manifest{}, oci.Image{}, fmt.Errorf("couldn't load image config: %s", err.Error())
	}

	if len(m.Layers) != len(c.RootFS.DiffIDs) {
		return oci.Manifest{}, oci.Image{}, fmt.Errorf("manifest + config mismatch: %d layers, %d diffIDs", len(m.Layers), len(c.RootFS.DiffIDs))
	}

	return m, c, nil
}

func (h *Handler) loadIndex() (oci.Index, error) {
	var i oci.Index
	if _, err := loadJSON(h.indexPath(), &i); err != nil {
		return oci.Index{}, err
	}

	if len(i.Manifests) != 1 {
		return oci.Index{}, fmt.Errorf("invalid # of manifests: expected 1, found %d", len(i.Manifests))
	}

	if i.Manifests[0].MediaType != oci.MediaTypeImageManifest {
		return oci.Index{}, fmt.Errorf("wrong media type for manifest: %s", i.Manifests[0].MediaType)
	}

	if i.Manifests[0].Platform != nil {
		return i, validatePlatform(i.Manifests[0].Platform.OS, i.Manifests[0].Platform.Architecture)
	}

	return i, nil
}

func (h *Handler) loadManifest(mDesc oci.Descriptor) (oci.Manifest, error) {
	var m oci.Manifest
	if err := h.loadDescriptor(mDesc, &m); err != nil {
		return oci.Manifest{}, err
	}

	if m.Config.MediaType != oci.MediaTypeImageConfig {
		return oci.Manifest{}, fmt.Errorf("wrong media type for image config: %s", m.Config.MediaType)
	}

	for _, layer := range m.Layers {
		if layer.MediaType != oci.MediaTypeImageLayerGzip {
			return oci.Manifest{}, fmt.Errorf("invalid layer media type: %s", layer.MediaType)
		}

		if err := h.validateSHA256(layer); err != nil {
			return oci.Manifest{}, fmt.Errorf("invalid layer: %s", err.Error())
		}
	}

	return m, nil
}

func (h *Handler) loadConfig(cDesc oci.Descriptor) (oci.Image, error) {
	var c oci.Image
	if err := h.loadDescriptor(cDesc, &c); err != nil {
		return oci.Image{}, err
	}

	if c.RootFS.Type != "layers" {
		return oci.Image{}, fmt.Errorf("invalid rootfs type: %s", c.RootFS.Type)
	}

	return c, validatePlatform(c.OS, c.Architecture)
}

func (h *Handler) loadDescriptor(desc oci.Descriptor, obj interface{}) error {
	expectedSha := desc.Digest.Encoded()

	sha, err := loadJSON(h.blobsPath(expectedSha), obj)
	if err != nil {
		return err
	}

	if sha != expectedSha {
		return fmt.Errorf("sha256 mismatch: expected %s, found %s", expectedSha, sha)

	}
	return nil
}

func (h *Handler) validateSHA256(d oci.Descriptor) error {
	expectedSha := d.Digest.Encoded()

	f, err := os.Open(h.blobsPath(expectedSha))
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}

	sha := fmt.Sprintf("%x", hash.Sum(nil))
	if sha != expectedSha {
		return fmt.Errorf("sha256 mismatch: expected %s, found %s", expectedSha, sha)
	}

	return nil
}

func loadJSON(file string, obj interface{}) (string, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(contents)), json.Unmarshal(contents, obj)
}

func validatePlatform(os string, arch string) error {
	if os != "windows" || arch != "amd64" {
		return fmt.Errorf("invalid platform: expected windows/amd64, found %s/%s", os, arch)
	}
	return nil
}
