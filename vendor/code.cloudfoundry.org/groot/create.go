package groot

import (
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	runspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

func (g *Groot) Create(handle string, rootfsURI *url.URL) (runspec.Spec, error) {
	g.Logger = g.Logger.Session("create")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	image, err := g.ImagePuller.Pull(g.Logger, imagepuller.ImageSpec{ImageSrc: rootfsURI})
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "pulling image")
	}

	bundle, err := g.Driver.Bundle(g.Logger.Session("bundle"), handle, image.ChainIDs)
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "creating bundle")
	}

	return bundle, nil
}
