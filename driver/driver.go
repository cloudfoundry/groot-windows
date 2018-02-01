package driver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/hcs"
	"code.cloudfoundry.org/groot-windows/privilege"
	"code.cloudfoundry.org/groot-windows/tarstream"
	"github.com/Microsoft/go-winio/archive/tar"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/hcsshim"
)

//go:generate counterfeiter -o fakes/tarstreamer.go --fake-name TarStreamer . TarStreamer
type TarStreamer interface {
	SetReader(io.Reader)
	Next() (*tar.Header, error)
	FileInfoFromHeader(*tar.Header) (string, int64, *winio.FileBasicInfo, error)
	WriteBackupStreamFromTarFile(io.Writer, *tar.Header) (*tar.Header, error)
}

//go:generate counterfeiter -o fakes/hcs_client.go --fake-name HCSClient . HCSClient
type HCSClient interface {
	NewLayerWriter(hcsshim.DriverInfo, string, []string) (hcs.LayerWriter, error)
	CreateLayer(hcsshim.DriverInfo, string, string, []string) error
	LayerExists(hcsshim.DriverInfo, string) (bool, error)
	GetLayerMountPath(hcsshim.DriverInfo, string) (string, error)
	DestroyLayer(hcsshim.DriverInfo, string) error
}

//go:generate counterfeiter -o fakes/privilege_elevator.go --fake-name PrivilegeElevator . PrivilegeElevator
type PrivilegeElevator interface {
	EnableProcessPrivileges([]string) error
	DisableProcessPrivileges([]string) error
}

const (
	LayerDir  = "layers"
	VolumeDir = "volumes"
)

type Creator struct{}

func (c *Creator) NewDriver(conf groot.Config) (groot.Driver, error) {
	if conf.Store == "" {
		return nil, errors.New("must set store")
	}

	layerStore := filepath.Join(conf.Store, LayerDir)
	if err := os.MkdirAll(layerStore, 0755); err != nil {
		return nil, fmt.Errorf("couldn't create layer store: %s", err.Error())
	}

	volumeStore := filepath.Join(conf.Store, VolumeDir)
	if err := os.MkdirAll(volumeStore, 0755); err != nil {
		return nil, fmt.Errorf("couldn't create volume store: %s", err.Error())
	}

	return New(layerStore, volumeStore, hcs.NewClient(), tarstream.New(), &privilege.Elevator{}), nil
}

type Driver struct {
	layerStore        string
	volumeStore       string
	hcsClient         HCSClient
	tarStreamer       TarStreamer
	privilegeElevator PrivilegeElevator
}

func New(layerStore, volumeStore string, hcsClient HCSClient, tarStreamer TarStreamer, privilegeElevator PrivilegeElevator) *Driver {
	return &Driver{
		layerStore:        layerStore,
		volumeStore:       volumeStore,
		hcsClient:         hcsClient,
		tarStreamer:       tarStreamer,
		privilegeElevator: privilegeElevator,
	}
}

func (d *Driver) LayerStore() string {
	return d.layerStore
}

func (d *Driver) VolumeStore() string {
	return d.volumeStore
}
