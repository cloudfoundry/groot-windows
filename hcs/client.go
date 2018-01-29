package hcs

import (
	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/hcsshim"
)

//go:generate counterfeiter -o fakes/layer_writer.go --fake-name LayerWriter . LayerWriter
// LayerWriter is an interface that supports writing a new container image layer.
type LayerWriter interface {
	// Add adds a file to the layer with given metadata.
	Add(name string, fileInfo *winio.FileBasicInfo) error
	// AddLink adds a hard link to the layer. The target must already have been added.
	AddLink(name string, target string) error
	// Remove removes a file that was present in a parent layer from the layer.
	Remove(name string) error
	// Write writes data to the current file. The data must be in the format of a Win32
	// backup stream.
	Write(b []byte) (int, error)
	// Close finishes the layer writing process and releases any resources.
	Close() error
}

type Client struct{}

func (c *Client) NewLayerWriter(info hcsshim.DriverInfo, layerID string, parentLayerPaths []string) (LayerWriter, error) {
	return hcsshim.NewLayerWriter(info, layerID, parentLayerPaths)
}
