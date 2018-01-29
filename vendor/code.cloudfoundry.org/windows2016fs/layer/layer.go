package layer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Microsoft/hcsshim"
)

//go:generate counterfeiter . Writer
type Writer interface {
	WriteLayer(string) error
	SetHCSLayerWriter(hcsshim.DriverInfo, string, []string) error
}

type Manager struct {
	driverInfo hcsshim.DriverInfo
	writer     Writer
}

type State int

const (
	NotExist = iota
	Incomplete
	Valid
)

func NewManager(driverInfo hcsshim.DriverInfo, writer Writer) *Manager {
	return &Manager{
		driverInfo: driverInfo,
		writer:     writer,
	}
}

func (m *Manager) State(id string) (State, error) {
	layerDir := filepath.Join(m.driverInfo.HomeDir, id)
	_, err := os.Stat(layerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return NotExist, nil
		}

		return Incomplete, err
	}

	data, err := ioutil.ReadFile(filepath.Join(layerDir, ".complete"))
	if err != nil || string(data) != id {
		return Incomplete, nil
	}

	return Valid, nil
}

func (m *Manager) Delete(layerId string) error {
	return hcsshim.DestroyLayer(m.driverInfo, layerId)
}

func (m *Manager) Extract(layerGzipFile, layerId string, parentLayerIds []string) error {
	layerPath := filepath.Join(m.driverInfo.HomeDir, layerId)
	if err := os.MkdirAll(layerPath, 0755); err != nil {
		return err
	}

	parentLayerPaths := []string{}
	for _, id := range parentLayerIds {
		parentLayerPaths = append(parentLayerPaths, filepath.Join(m.driverInfo.HomeDir, id))
	}

	if err := m.writer.SetHCSLayerWriter(m.driverInfo, layerId, parentLayerPaths); err != nil {
		return fmt.Errorf("Failed to set up new layer writer: %s", err.Error())
	}

	if err := m.writer.WriteLayer(layerGzipFile); err != nil {
		return err
	}

	if len(parentLayerPaths) > 0 {
		data, err := json.Marshal(parentLayerPaths)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.Join(layerPath, "layerchain.json"), data, 0644); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(filepath.Join(layerPath, ".complete"), []byte(layerId), 0644)
}
