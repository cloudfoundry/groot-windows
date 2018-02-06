package groot

import (
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	runspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

type BundleSpec struct {
	DiskLimit             int64
	ExcludeImageFromQuota bool
	BaseImageSize         int64
}

func (g *Groot) Create(handle string, rootfsURI *url.URL, diskLimit int64, excludeImageFromQuota bool) (runspec.Spec, error) {
	g.Logger = g.Logger.Session("create")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	imageSpec := imagepuller.ImageSpec{
		ImageSrc:              rootfsURI,
		DiskLimit:             diskLimit,
		ExcludeImageFromQuota: excludeImageFromQuota,
	}

	image, err := g.ImagePuller.Pull(g.Logger, imageSpec)
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "pulling image")
	}

	bundleSpec := BundleSpec{
		DiskLimit:             diskLimit,
		ExcludeImageFromQuota: excludeImageFromQuota,
		BaseImageSize:         image.BaseImageSize,
	}

	bundle, err := g.Driver.Bundle(g.Logger.Session("bundle"), handle, image.ChainIDs, bundleSpec)
	if err != nil {
		return runspec.Spec{}, errors.Wrap(err, "creating bundle")
	}

	return bundle, nil
}
