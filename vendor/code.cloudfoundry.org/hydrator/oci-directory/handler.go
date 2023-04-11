package directory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	digest "github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

type Handler struct {
	ociImageDir string
}

func NewHandler(oid string) *Handler {
	return &Handler{
		/* handle both oci directory path and oci:///<directory-path> */
		ociImageDir: strings.TrimPrefix(oid, "oci:///"),
	}
}

func (h *Handler) AddBlob(srcBlobPath string, blobDescriptor oci.Descriptor) error {
	layerfd, err := os.Open(srcBlobPath)
	if err != nil {
		return err
	}
	defer layerfd.Close()

	if _, err := os.Stat(h.blobsDir()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is not a valid OCI image: %s directory missing", h.ociImageDir, h.blobsDir())
		}
		return err
	}

	if err := blobDescriptor.Digest.Validate(); err != nil {
		return err
	}

	destfd, err := os.Create(h.blobsPath(blobDescriptor.Digest.Encoded()))
	if err != nil {
		return err
	}
	defer destfd.Close()

	_, err = io.Copy(destfd, layerfd)
	return err
}

func (h *Handler) RemoveTopBlob(sha256 string) error {
	if _, err := os.Stat(h.blobsDir()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is not a valid OCI image: %s directory missing", h.ociImageDir, h.blobsDir())
		}
		return err
	}

	if err := os.Remove(h.blobsPath(sha256)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not contain layer: %s", h.ociImageDir, sha256)
		}
		return err
	}
	return nil
}

func (h *Handler) ClearMetadata() error {
	filesToDelete := []string{h.ociLayoutPath(), h.indexPath()}

	i, err := h.loadIndex()
	if err != nil {
		return fmt.Errorf("couldn't load index.json: %s", err.Error())
	}

	mDesc := i.Manifests[0]
	filesToDelete = append(filesToDelete, h.blobsPathFromDescriptor(mDesc))

	m, err := h.loadManifest(mDesc)
	if err != nil {
		return fmt.Errorf("couldn't load manifest: %s", err.Error())
	}
	filesToDelete = append(filesToDelete, h.blobsPathFromDescriptor(m.Config))

	var errRet error
	for _, f := range filesToDelete {
		if err := os.RemoveAll(f); err != nil {
			errRet = err
		}
	}

	return errRet
}

func (h *Handler) blobsPathFromDescriptor(desc oci.Descriptor) string {
	return h.blobsPath(desc.Digest.Encoded())
}

func (h *Handler) blobsPath(filename string) string {
	return filepath.Join(h.blobsDir(), filename)
}

func (h *Handler) blobsDir() string {
	return filepath.Join(h.ociImageDir, "blobs", string(digest.SHA256))
}

func (h *Handler) indexPath() string {
	return filepath.Join(h.ociImageDir, "index.json")
}

func (h *Handler) ociLayoutPath() string {
	return filepath.Join(h.ociImageDir, "oci-layout")
}
