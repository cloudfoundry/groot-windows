package downloader_test

import (
	"errors"
	"io/ioutil"

	"code.cloudfoundry.org/windows2016fs/downloader"
	"code.cloudfoundry.org/windows2016fs/downloader/downloaderfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Downloader", func() {
	const (
		downloadDir = "some-directory"
	)

	var (
		sourceLayers   []v1.Descriptor
		sourceDiffIds  []digest.Digest
		sourceConfig   v1.Image
		manifestConfig v1.Descriptor
		manifest       v1.Manifest
		registry       *downloaderfakes.FakeRegistry
		d              *downloader.Downloader
	)

	BeforeEach(func() {
		sourceLayers = []v1.Descriptor{
			{Digest: digest.NewDigestFromEncoded(digest.SHA256, "layer1"), Size: 1234},
			{Digest: digest.NewDigestFromEncoded(digest.SHA256, "layer2"), Size: 6789},
		}
		manifestConfig = v1.Descriptor{Digest: "config", Size: 7777}
		manifest = v1.Manifest{Layers: sourceLayers, Config: manifestConfig}

		sourceDiffIds = []digest.Digest{
			digest.NewDigestFromEncoded(digest.SHA256, "aaaaaa"),
			digest.NewDigestFromEncoded(digest.SHA256, "bbbbbb"),
		}
		sourceConfig = v1.Image{
			OS:           "windows",
			Architecture: "amd64",
			RootFS:       v1.RootFS{Type: "layers", DiffIDs: sourceDiffIds},
		}

		registry = &downloaderfakes.FakeRegistry{}

		registry.ManifestReturnsOnCall(0, manifest, nil)
		registry.ConfigReturnsOnCall(0, sourceConfig, nil)

		d = downloader.New(downloadDir, registry, ioutil.Discard)
	})

	Describe("Run", func() {
		It("Uses the manifest to download the config, all the layers and returns the proper descriptors + diffIds", func() {
			layers, diffIds, err := d.Run()
			Expect(err).NotTo(HaveOccurred())

			Expect(layers[0].Digest).To(Equal(digest.Digest("sha256:layer1")))
			Expect(layers[0].Size).To(Equal(int64(1234)))
			Expect(layers[0].MediaType).To(Equal(v1.MediaTypeImageLayerGzip))

			Expect(layers[1].Digest).To(Equal(digest.Digest("sha256:layer2")))
			Expect(layers[1].Size).To(Equal(int64(6789)))
			Expect(layers[1].MediaType).To(Equal(v1.MediaTypeImageLayerGzip))

			Expect(diffIds).To(ConsistOf(sourceDiffIds))

			Expect(registry.ManifestCallCount()).To(Equal(1))
			Expect(registry.ConfigCallCount()).To(Equal(1))
			Expect(registry.ConfigArgsForCall(0)).To(Equal(manifestConfig))

			Expect(registry.DownloadLayerCallCount()).To(Equal(2))
			l1, dir := registry.DownloadLayerArgsForCall(0)
			Expect(dir).To(Equal("some-directory"))

			l2, dir := registry.DownloadLayerArgsForCall(1)
			Expect(dir).To(Equal("some-directory"))

			Expect([]v1.Descriptor{l1, l2}).To(ConsistOf(sourceLayers))
		})
	})

	Context("getting the manifest fails", func() {
		BeforeEach(func() {
			registry.ManifestReturnsOnCall(0, v1.Manifest{}, errors.New("couldn't get manifest"))
		})

		It("returns an error", func() {
			_, _, err := d.Run()
			Expect(err.Error()).To(Equal("couldn't get manifest"))
			Expect(registry.DownloadLayerCallCount()).To(Equal(0))
		})
	})

	Context("manifest and config have different # of layers", func() {
		BeforeEach(func() {
			sourceDiffIds = []digest.Digest{
				digest.NewDigestFromEncoded(digest.SHA256, "aaaaaa"),
			}
			sourceConfig = v1.Image{
				OS:           "windows",
				Architecture: "amd64",
				RootFS:       v1.RootFS{Type: "layers", DiffIDs: sourceDiffIds},
			}

			registry.ConfigReturnsOnCall(0, sourceConfig, nil)
		})

		It("returns an error", func() {
			_, _, err := d.Run()
			Expect(err.Error()).To(Equal("mismatch: 2 layers, 1 diffIds"))
			Expect(registry.DownloadLayerCallCount()).To(Equal(0))
		})
	})

	Context("config has invalid OS", func() {
		BeforeEach(func() {
			sourceConfig = v1.Image{
				OS: "linux",
			}

			registry.ConfigReturnsOnCall(0, sourceConfig, nil)
		})

		It("returns an error", func() {
			_, _, err := d.Run()
			Expect(err.Error()).To(Equal("invalid container OS: linux"))
			Expect(registry.DownloadLayerCallCount()).To(Equal(0))
		})
	})

	Context("config has invalid architecture", func() {
		BeforeEach(func() {
			sourceConfig = v1.Image{
				OS:           "windows",
				Architecture: "ppc64",
			}

			registry.ConfigReturnsOnCall(0, sourceConfig, nil)
		})

		It("returns an error", func() {
			_, _, err := d.Run()
			Expect(err.Error()).To(Equal("invalid container arch: ppc64"))
			Expect(registry.DownloadLayerCallCount()).To(Equal(0))
		})
	})

	Context("downloading a layer fails", func() {
		BeforeEach(func() {
			registry.DownloadLayerReturnsOnCall(1, errors.New("couldn't download layer2"))
		})

		It("returns an error", func() {
			_, _, err := d.Run()
			Expect(err.Error()).To(Equal("couldn't download layer2"))
		})
	})
})
