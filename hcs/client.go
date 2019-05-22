package hcs

import (
	"fmt"

	"code.cloudfoundry.org/filelock"
	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/hcsshim"
)

//go:generate counterfeiter -o fakes/layer_writer.go --fake-name LayerWriter . LayerWriter
type LayerWriter interface {
	Add(name string, fileInfo *winio.FileBasicInfo) error
	AddLink(name string, target string) error
	Remove(name string) error
	Write(b []byte) (int, error)
	Close() error
}

type Client struct {
	layerCreateLock filelock.FileLocker
}

func NewClient() *Client {
	return &Client{
		layerCreateLock: filelock.NewLocker("C:\\var\\vcap\\data\\groot-windows\\create.lock"),
	}
}

func (c *Client) NewLayerWriter(di hcsshim.DriverInfo, layerID string, parentLayerPaths []string) (LayerWriter, error) {
	return hcsshim.NewLayerWriter(di, layerID, parentLayerPaths)
}

func (c *Client) GetLayerMountPath(di hcsshim.DriverInfo, id string) (string, error) {
	return hcsshim.GetLayerMountPath(di, id)
}

func (c *Client) CreateLayer(di hcsshim.DriverInfo, id string, parentLayerPaths []string) error {
	f, err := c.layerCreateLock.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	if err := hcsshim.CreateSandboxLayer(di, id, "", parentLayerPaths); err != nil {
		return err
	}

	if err := hcsshim.ActivateLayer(di, id); err != nil {
		return err
	}

	return hcsshim.PrepareLayer(di, id, parentLayerPaths)
}

func (c *Client) DestroyLayer(di hcsshim.DriverInfo, id string) error {
	var unprepareErr, deactivateErr, destroyErr error

	for i := 0; i < 3; i++ {
		unprepareErr = hcsshim.UnprepareLayer(di, id)
		deactivateErr = hcsshim.DeactivateLayer(di, id)
		destroyErr = hcsshim.DestroyLayer(di, id)
		if destroyErr == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to remove layer (unprepare error: %s, deactivate error: %s, destroy error: %s)", unprepareErr.Error(), deactivateErr.Error(), destroyErr.Error())
}

func (c *Client) LayerExists(di hcsshim.DriverInfo, id string) (bool, error) {
	return hcsshim.LayerExists(di, id)
}
