package driver

import (
	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (d *Driver) Bundle(logger lager.Logger, bundleID string, layerIDs []string) (specs.Spec, error) {
	return specs.Spec{}, nil
}
