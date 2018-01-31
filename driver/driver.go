package driver

import (
	"io"

	"code.cloudfoundry.org/groot-windows/hcs"
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
