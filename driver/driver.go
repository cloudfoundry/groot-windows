package driver

import (
	"io"
	"path/filepath"

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
	CreateLayer(hcsshim.DriverInfo, string, []string) error
	LayerExists(hcsshim.DriverInfo, string) (bool, error)
	GetLayerMountPath(hcsshim.DriverInfo, string) (string, error)
	DestroyLayer(hcsshim.DriverInfo, string) error
}

//go:generate counterfeiter -o fakes/privilege_elevator.go --fake-name PrivilegeElevator . PrivilegeElevator
type PrivilegeElevator interface {
	EnableProcessPrivileges([]string) error
	DisableProcessPrivileges([]string) error
}

//go:generate counterfeiter -o fakes/limiter.go --fake-name Limiter . Limiter
type Limiter interface {
	SetQuota(string, uint64) error
	GetQuotaUsed(string) (uint64, error)
}

const (
	layerDir  = "layers"
	volumeDir = "volumes"
)

type Driver struct {
	Store             string
	hcsClient         HCSClient
	tarStreamer       TarStreamer
	privilegeElevator PrivilegeElevator
	limiter           Limiter
}

func New(hcsClient HCSClient, tarStreamer TarStreamer, privilegeElevator PrivilegeElevator, limiter Limiter) *Driver {
	return &Driver{
		hcsClient:         hcsClient,
		tarStreamer:       tarStreamer,
		privilegeElevator: privilegeElevator,
		limiter:           limiter,
	}
}

func (d *Driver) LayerStore() string {
	return toWindowsPath(filepath.Join(d.Store, layerDir))
}

func (d *Driver) VolumeStore() string {
	return toWindowsPath(filepath.Join(d.Store, volumeDir))
}

func (d *Driver) metadataFile(bundleId string) string {
	return filepath.Join(d.VolumeStore(), bundleId, "metadata.json")
}

func (d *Driver) layerSizeFile(layerId string) string {
	return filepath.Join(d.LayerStore(), layerId, "size")
}

func toWindowsPath(input string) string {
	vol := filepath.VolumeName(input)
	if vol == "" {
		input = filepath.Join("C:\\", input)
	}
	return filepath.Clean(input)
}
