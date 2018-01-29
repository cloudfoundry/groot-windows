package imagepuller_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/groot/imagepuller/imagepullerfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Image Puller", func() {
	var (
		logger           lager.Logger
		fakeFetcher      *imagepullerfakes.FakeFetcher
		fakeVolumeDriver *imagepullerfakes.FakeVolumeDriver
		expectedImgDesc  specsv1.Image

		imagePuller *imagepuller.ImagePuller
		layerInfos  []imagepuller.LayerInfo

		imageSrcURL   *url.URL
		tmpVolumesDir string
	)

	BeforeEach(func() {
		fakeFetcher = new(imagepullerfakes.FakeFetcher)
		expectedImgDesc = specsv1.Image{Author: "Groot"}
		layerInfos = []imagepuller.LayerInfo{
			{BlobID: "i-am-a-layer", ChainID: "layer-111", ParentChainID: ""},
			{BlobID: "i-am-another-layer", ChainID: "chain-222", ParentChainID: "layer-111"},
			{BlobID: "i-am-the-last-layer", ChainID: "chain-333", ParentChainID: "chain-222"},
		}
		fakeFetcher.ImageInfoReturns(
			imagepuller.ImageInfo{
				LayerInfos: layerInfos,
				Config:     expectedImgDesc,
			}, nil)

		fakeFetcher.StreamBlobStub = func(_ lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
			buffer := bytes.NewBuffer([]byte{})
			stream := gzip.NewWriter(buffer)
			defer stream.Close()
			return ioutil.NopCloser(buffer), 0, nil
		}

		var err error
		tmpVolumesDir, err = ioutil.TempDir("", "volumes")
		Expect(err).NotTo(HaveOccurred())

		fakeVolumeDriver = new(imagepullerfakes.FakeVolumeDriver)
		imagePuller = imagepuller.NewImagePuller(fakeFetcher, fakeVolumeDriver)
		logger = lagertest.NewTestLogger("image-puller")

		imageSrcURL, err = url.Parse("docker:///an/image")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the image description", func() {
		image, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
			ImageSrc: imageSrcURL,
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(image.Image).To(Equal(expectedImgDesc))
	})

	It("returns the chain ids in the order specified by the image", func() {
		image, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
			ImageSrc: imageSrcURL,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(image.ChainIDs).To(Equal([]string{"layer-111", "chain-222", "chain-333"}))
	})

	It("passes the correct parentIDs to Unpack", func() {
		_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
			ImageSrc: imageSrcURL,
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(3))

		_, _, parentIDs, _ := fakeVolumeDriver.UnpackArgsForCall(0)
		Expect(parentIDs).To(BeEmpty())
		_, _, parentIDs, _ = fakeVolumeDriver.UnpackArgsForCall(1)
		Expect(parentIDs).To(Equal([]string{"layer-111"}))
		_, _, parentIDs, _ = fakeVolumeDriver.UnpackArgsForCall(2)
		Expect(parentIDs).To(Equal([]string{"layer-111", "chain-222"}))
	})

	It("unpacks the layers got from the fetcher", func() {
		fakeFetcher.StreamBlobStub = func(_ lager.Logger, imageURL *url.URL, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
			Expect(imageURL).To(Equal(imageSrcURL))

			buffer := bytes.NewBuffer([]byte{})
			stream := gzip.NewWriter(buffer)
			defer stream.Close()
			_, err := stream.Write([]byte(fmt.Sprintf("layer-%s-contents", layerInfo.BlobID)))
			Expect(err).NotTo(HaveOccurred())
			return ioutil.NopCloser(buffer), 1200, nil
		}

		_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
			ImageSrc: imageSrcURL,
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(3))

		validateLayer := func(idx int, expected string) {
			_, _, _, stream := fakeVolumeDriver.UnpackArgsForCall(idx)
			gzipReader, err := gzip.NewReader(stream)
			Expect(err).NotTo(HaveOccurred())
			contents, err := ioutil.ReadAll(gzipReader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal(expected))
		}

		validateLayer(0, "layer-i-am-a-layer-contents")
		validateLayer(1, "layer-i-am-another-layer-contents")
		validateLayer(2, "layer-i-am-the-last-layer-contents")
	})

	Context("when the layers size in the manifest will exceed the limit", func() {
		Context("when including the image size in the limit", func() {
			It("returns an error", func() {
				fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
					LayerInfos: []imagepuller.LayerInfo{
						{Size: 1000},
						{Size: 201},
					},
				}, nil)

				_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
					ImageSrc:              imageSrcURL,
					DiskLimit:             1200,
					ExcludeImageFromQuota: false,
				})
				Expect(err).To(MatchError(ContainSubstring("layers exceed disk quota")))
			})

			Context("when the disk limit is zero", func() {
				It("doesn't fail", func() {
					fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
						LayerInfos: []imagepuller.LayerInfo{
							{Size: 1000},
							{Size: 201},
						},
					}, nil)

					_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
						ImageSrc:              imageSrcURL,
						DiskLimit:             0,
						ExcludeImageFromQuota: false,
					})

					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when not including the image size in the limit", func() {
			It("doesn't fail", func() {
				fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
					LayerInfos: []imagepuller.LayerInfo{
						{Size: 1000},
						{Size: 201},
					},
				}, nil)

				_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
					ImageSrc:              imageSrcURL,
					DiskLimit:             1024,
					ExcludeImageFromQuota: true,
				})

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("when fetching the list of layers fails", func() {
		BeforeEach(func() {
			fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
				LayerInfos: []imagepuller.LayerInfo{},
				Config:     specsv1.Image{},
			}, errors.New("failed to get list of layers"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
				ImageSrc: imageSrcURL,
			})
			Expect(err).To(MatchError(ContainSubstring("failed to get list of layers")))
		})
	})

	Context("when all volumes exist", func() {
		BeforeEach(func() {
			fakeVolumeDriver.ExistsReturns(true)
		})

		It("does not try to unpack any layer", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
				ImageSrc: imageSrcURL,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(0))
		})
	})

	Context("when one volume exists", func() {
		BeforeEach(func() {
			fakeVolumeDriver.ExistsStub = func(_ lager.Logger, id string) bool {
				if id == "chain-222" {
					return true
				}
				return false
			}
		})

		It("only creates the children of the existing volume", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
				ImageSrc: imageSrcURL,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(1))
			_, volID, _, _ := fakeVolumeDriver.UnpackArgsForCall(0)
			Expect(volID).To(Equal("chain-333"))
		})
	})

	Context("when creating a volume fails", func() {
		BeforeEach(func() {
			fakeVolumeDriver.UnpackReturns(errors.New("failed to create volume"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
				ImageSrc: imageSrcURL,
			})
			Expect(err).To(MatchError(ContainSubstring("failed to create volume")))
		})
	})

	Context("when streaming a blob fails", func() {
		BeforeEach(func() {
			fakeFetcher.StreamBlobReturns(nil, 0, errors.New("failed to stream blob"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{ImageSrc: imageSrcURL})
			Expect(err).To(MatchError(ContainSubstring("failed to stream blob")))
		})
	})

	Context("when unpacking a blob fails", func() {
		BeforeEach(func() {
			count := 0
			fakeVolumeDriver.UnpackStub = func(_ lager.Logger, id string, parentIDs []string, stream io.Reader) error {
				count++
				if count == 3 {
					return errors.New("failed to unpack the blob")
				}

				return nil
			}
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{ImageSrc: imageSrcURL})
			Expect(err).To(MatchError(ContainSubstring("failed to unpack the blob")))
		})
	})
})
