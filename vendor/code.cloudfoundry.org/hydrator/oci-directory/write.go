package directory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

func (h *Handler) WriteMetadata(layers []oci.Descriptor, diffIds []digest.Digest, layerAdded bool) error {
	if err := h.writeOCILayout(); err != nil {
		return err
	}

	configDescriptor, err := h.writeConfig(diffIds)
	if err != nil {
		return err
	}

	annotations := make(map[string]string)
	/* Mark that the top layer was added using hydrator */
	if layerAdded == true {
		annotations["hydrator.layerAdded"] = "true"
	}

	manifestDescriptor, err := h.writeManifest(layers, configDescriptor, annotations)
	if err != nil {
		return err
	}

	return h.writeIndexJson(manifestDescriptor)
}

func (h *Handler) writeOCILayout() error {
	il := oci.ImageLayout{
		Version: specs.Version,
	}
	data, err := json.Marshal(il)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(h.ociLayoutPath(), data, 0644)
}

func (h *Handler) writeConfig(diffIds []digest.Digest) (oci.Descriptor, error) {
	ic := oci.Image{
		Architecture: "amd64",
		OS:           "windows",
		RootFS:       oci.RootFS{Type: "layers", DiffIDs: diffIds},
	}

	d, err := h.writeBlob(ic)
	if err != nil {
		return oci.Descriptor{}, err
	}

	d.MediaType = oci.MediaTypeImageConfig
	return d, nil
}

func (h *Handler) writeManifest(layers []oci.Descriptor, config oci.Descriptor, annotations map[string]string) (oci.Descriptor, error) {
	im := oci.Manifest{
		Versioned:   specs.Versioned{SchemaVersion: 2},
		Config:      config,
		Layers:      layers,
		Annotations: annotations,
	}

	d, err := h.writeBlob(im)
	if err != nil {
		return oci.Descriptor{}, err
	}

	d.MediaType = oci.MediaTypeImageManifest
	d.Platform = &oci.Platform{OS: "windows", Architecture: "amd64"}
	return d, nil
}

func (h *Handler) writeBlob(blob interface{}) (oci.Descriptor, error) {
	data, err := json.Marshal(blob)
	if err != nil {
		return oci.Descriptor{}, err
	}

	if err := os.MkdirAll(h.blobsDir(), 0755); err != nil {
		return oci.Descriptor{}, err
	}

	blobSha := fmt.Sprintf("%x", sha256.Sum256(data))
	blobFile := h.blobsPath(blobSha)

	if err := ioutil.WriteFile(blobFile, data, 0644); err != nil {
		return oci.Descriptor{}, err
	}

	return oci.Descriptor{
		Size:   int64(len(data)),
		Digest: digest.NewDigestFromEncoded(digest.SHA256, blobSha),
	}, nil
}

func (h *Handler) writeIndexJson(manifest oci.Descriptor) error {
	ii := oci.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: []oci.Descriptor{manifest},
	}

	data, err := json.Marshal(ii)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(h.indexPath(), data, 0644)
}
