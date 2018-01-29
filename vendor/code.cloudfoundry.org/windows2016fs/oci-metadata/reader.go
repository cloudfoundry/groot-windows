package metadata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

type Reader struct {
	srcDir string
}

func NewReader(srcDir string) *Reader {
	return &Reader{
		srcDir: srcDir,
	}
}

func (r *Reader) Read() (v1.Manifest, v1.Image, error) {
	i, err := r.loadIndex()
	if err != nil {
		return v1.Manifest{}, v1.Image{}, fmt.Errorf("couldn't load index.json: %s", err.Error())
	}

	mDesc := i.Manifests[0]
	m, err := r.loadManifest(mDesc)
	if err != nil {
		return v1.Manifest{}, v1.Image{}, fmt.Errorf("couldn't load manifest: %s", err.Error())
	}

	c, err := r.loadConfig(m.Config)
	if err != nil {
		return v1.Manifest{}, v1.Image{}, fmt.Errorf("couldn't load image config: %s", err.Error())
	}

	if len(m.Layers) != len(c.RootFS.DiffIDs) {
		return v1.Manifest{}, v1.Image{}, fmt.Errorf("manifest + config mismatch: %d layers, %d diffIDs", len(m.Layers), len(c.RootFS.DiffIDs))
	}

	return m, c, nil
}

func (r *Reader) loadIndex() (v1.Index, error) {
	var i v1.Index
	if _, err := loadJSON(filepath.Join(r.srcDir, "index.json"), &i); err != nil {
		return v1.Index{}, err
	}

	if len(i.Manifests) != 1 {
		return v1.Index{}, fmt.Errorf("invalid # of manifests: expected 1, found %d", len(i.Manifests))
	}

	if i.Manifests[0].MediaType != v1.MediaTypeImageManifest {
		return v1.Index{}, fmt.Errorf("wrong media type for manifest: %s", i.Manifests[0].MediaType)
	}

	if i.Manifests[0].Platform != nil {
		return i, validatePlatform(i.Manifests[0].Platform.OS, i.Manifests[0].Platform.Architecture)
	}

	return i, nil
}

func (r *Reader) loadManifest(mDesc v1.Descriptor) (v1.Manifest, error) {
	var m v1.Manifest
	if err := r.loadDescriptor(mDesc, &m); err != nil {
		return v1.Manifest{}, err
	}

	if m.Config.MediaType != v1.MediaTypeImageConfig {
		return v1.Manifest{}, fmt.Errorf("wrong media type for image config: %s", m.Config.MediaType)
	}

	for _, layer := range m.Layers {
		if layer.MediaType != v1.MediaTypeImageLayerGzip {
			return v1.Manifest{}, fmt.Errorf("invalid layer media type: %s", layer.MediaType)
		}

		if err := r.validateSHA256(layer); err != nil {
			return v1.Manifest{}, fmt.Errorf("invalid layer: %s", err.Error())
		}
	}

	return m, nil
}

func (r *Reader) loadConfig(cDesc v1.Descriptor) (v1.Image, error) {
	var c v1.Image
	if err := r.loadDescriptor(cDesc, &c); err != nil {
		return v1.Image{}, err
	}

	if c.RootFS.Type != "layers" {
		return v1.Image{}, fmt.Errorf("invalid rootfs type: %s", c.RootFS.Type)
	}

	return c, validatePlatform(c.OS, c.Architecture)
}

func (r *Reader) loadDescriptor(desc v1.Descriptor, obj interface{}) error {
	expectedSha := desc.Digest.Encoded()

	sha, err := loadJSON(filepath.Join(r.srcDir, "blobs", "sha256", expectedSha), obj)
	if err != nil {
		return err
	}

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

func (r *Reader) validateSHA256(d v1.Descriptor) error {
	expectedSha := d.Digest.Encoded()
	file := filepath.Join(r.srcDir, "blobs", "sha256", expectedSha)

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	sha := fmt.Sprintf("%x", h.Sum(nil))
	if sha != expectedSha {
		return fmt.Errorf("sha256 mismatch: expected %s, found %s", expectedSha, sha)
	}

	return nil
}
