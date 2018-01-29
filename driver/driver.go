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
	NewLayerWriter(info hcsshim.DriverInfo, layerID string, parentLayerPaths []string) (hcs.LayerWriter, error)
}

//go:generate counterfeiter -o fakes/privilege_elevator.go --fake-name PrivilegeElevator . PrivilegeElevator
type PrivilegeElevator interface {
	EnableProcessPrivileges([]string) error
	DisableProcessPrivileges([]string) error
}

type Driver struct {
	layerStore        string
	hcsClient         HCSClient
	tarStreamer       TarStreamer
	privilegeElevator PrivilegeElevator
}

func New(layerStore string, hcsClient HCSClient, tarStreamer TarStreamer, privilegeElevator PrivilegeElevator) *Driver {
	return &Driver{
		layerStore:        layerStore,
		hcsClient:         hcsClient,
		tarStreamer:       tarStreamer,
		privilegeElevator: privilegeElevator,
	}
}
