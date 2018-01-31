package groot

import (
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	"github.com/pkg/errors"
)

func (g *Groot) Pull(rootfsURI *url.URL) error {
	g.Logger = g.Logger.Session("pull")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	_, err := g.ImagePuller.Pull(g.Logger, imagepuller.ImageSpec{ImageSrc: rootfsURI})
	return errors.Wrap(err, "pulling image")
}
