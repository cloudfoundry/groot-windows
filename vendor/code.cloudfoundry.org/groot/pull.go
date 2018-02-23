package groot

import (
	"code.cloudfoundry.org/groot/imagepuller"
	"github.com/pkg/errors"
)

func (g *Groot) Pull() error {
	g.Logger = g.Logger.Session("pull")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	_, err := g.ImagePuller.Pull(g.Logger, imagepuller.ImageSpec{})
	return errors.Wrap(err, "pulling image")
}
