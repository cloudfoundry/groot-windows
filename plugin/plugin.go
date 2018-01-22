package plugin

import (
	"io"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Plugin struct{}

func (p *Plugin) Unpack(logger lager.Logger, layerID, parentID string, layerTar io.Reader) error {
	return nil
}
func (p *Plugin) Bundle(logger lager.Logger, bundleID string, layerIDs []string) (specs.Spec, error) {
	return specs.Spec{}, nil
}
func (p *Plugin) Exists(logger lager.Logger, layerID string) bool {
	return false
}
func (p *Plugin) Delete(logger lager.Logger, bundleID string) error {
	return nil
}
