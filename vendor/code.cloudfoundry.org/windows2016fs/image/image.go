package image

import (
	"fmt"
	"io"
	"path/filepath"

	"code.cloudfoundry.org/windows2016fs/layer"

	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

//go:generate counterfeiter . LayerManager
type LayerManager interface {
	Extract(string, string, []string) error
	Delete(string) error
	State(string) (layer.State, error)
}

//go:generate counterfeiter . MetadataReader
type MetadataReader interface {
	Read() (v1.Manifest, v1.Image, error)
}

type Manager struct {
	srcDir         string
	metadataReader MetadataReader
	layerManager   LayerManager
	output         io.Writer
}

func NewManager(srcDir string, metadataReader MetadataReader, layerManager LayerManager, output io.Writer) *Manager {
	return &Manager{
		srcDir:         srcDir,
		metadataReader: metadataReader,
		layerManager:   layerManager,
		output:         output,
	}
}

func (m *Manager) Extract() (string, error) {
	parentLayerIds := []string{}

	manifest, config, err := m.metadataReader.Read()
	if err != nil {
		return "", err
	}

	for i, l := range manifest.Layers {
		layerSHA, err := digestSHA(l.Digest)
		if err != nil {
			return "", err
		}
		layerTgz := filepath.Join(m.srcDir, "blobs", "sha256", layerSHA)

		layerId, err := digestSHA(config.RootFS.DiffIDs[i])
		if err != nil {
			return "", err
		}

		state, err := m.layerManager.State(layerId)
		if err != nil {
			return "", err
		}

		switch state {
		case layer.Incomplete:
			if err := m.layerManager.Delete(layerId); err != nil {
				return "", err
			}
			fallthrough
		case layer.NotExist:
			fmt.Fprintf(m.output, "Extracting %s... ", layerId)
			if err := m.layerManager.Extract(layerTgz, layerId, parentLayerIds); err != nil {
				return "", err
			}
			fmt.Fprintln(m.output, "Done.")
		case layer.Valid:
			fmt.Fprintf(m.output, "Layer %s already exists\n", layerId)
		default:
			panic(fmt.Sprintf("invalid layer state: %d", state))
		}

		parentLayerIds = append([]string{layerId}, parentLayerIds...)
	}

	return parentLayerIds[0], nil
}

func digestSHA(d digest.Digest) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}

	return d.Encoded(), nil
}
