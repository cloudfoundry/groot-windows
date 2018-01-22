package fetcher_test

import (
	"net/url"

	fetcherpkg "code.cloudfoundry.org/groot/fetcher"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/groot/imagepuller/imagepullerfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fetcher", func() {
	var (
		fileFetcher  *imagepullerfakes.FakeFetcher
		layerFetcher *imagepullerfakes.FakeFetcher
		fetcher      fetcherpkg.Fetcher
		logger       lager.Logger
		imageURL     *url.URL
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("image-puller")
		fileFetcher = new(imagepullerfakes.FakeFetcher)
		layerFetcher = new(imagepullerfakes.FakeFetcher)

		fetcher = fetcherpkg.Fetcher{
			FileFetcher:  fileFetcher,
			LayerFetcher: layerFetcher,
		}
	})

	Context("when the rootfsURI contains the docker scheme", func() {
		BeforeEach(func() {
			var err error
			imageURL, err = url.Parse("docker:///hello")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("ImageInfo", func() {
			It("calls the LayerFetcher", func() {
				fetcher.ImageInfo(logger, imageURL)

				Expect(layerFetcher.ImageInfoCallCount()).To(Equal(1))
				Expect(fileFetcher.ImageInfoCallCount()).To(Equal(0))
			})
		})

		Describe("StreamBlob", func() {
			It("calls the LayerFetcher", func() {
				fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})

				Expect(layerFetcher.StreamBlobCallCount()).To(Equal(1))
				Expect(fileFetcher.StreamBlobCallCount()).To(Equal(0))
			})
		})
	})

	Context("when the rootfsURI contains the oci scheme", func() {
		BeforeEach(func() {
			var err error
			imageURL, err = url.Parse("oci:///hello")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("ImageInfo", func() {
			It("calls the LayerFetcher", func() {
				fetcher.ImageInfo(logger, imageURL)

				Expect(layerFetcher.ImageInfoCallCount()).To(Equal(1))
				Expect(fileFetcher.ImageInfoCallCount()).To(Equal(0))
			})
		})

		Describe("StreamBlob", func() {
			It("calls the LayerFetcher", func() {
				fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})

				Expect(layerFetcher.StreamBlobCallCount()).To(Equal(1))
				Expect(fileFetcher.StreamBlobCallCount()).To(Equal(0))
			})
		})
	})

	Context("when the rootfsURI doesn't contain a scheme", func() {
		BeforeEach(func() {
			var err error
			imageURL, err = url.Parse("/hello")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("ImageInfo", func() {
			It("calls the FileFetcher", func() {
				fetcher.ImageInfo(logger, imageURL)

				Expect(fileFetcher.ImageInfoCallCount()).To(Equal(1))
				Expect(layerFetcher.ImageInfoCallCount()).To(Equal(0))
			})
		})

		Describe("StreamBlob", func() {
			It("calls the FileFetcher", func() {
				fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})

				Expect(fileFetcher.StreamBlobCallCount()).To(Equal(1))
				Expect(layerFetcher.StreamBlobCallCount()).To(Equal(0))
			})
		})
	})

	Context("when the rootfsURI contains a drive letter", func() {
		BeforeEach(func() {
			var err error
			imageURL, err = url.Parse("c:/hello")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("ImageInfo", func() {
			It("calls the FileFetcher", func() {
				fetcher.ImageInfo(logger, imageURL)

				Expect(fileFetcher.ImageInfoCallCount()).To(Equal(1))
				Expect(layerFetcher.ImageInfoCallCount()).To(Equal(0))
			})
		})

		Describe("StreamBlob", func() {
			It("calls the FileFetcher", func() {
				fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})

				Expect(fileFetcher.StreamBlobCallCount()).To(Equal(1))
				Expect(layerFetcher.StreamBlobCallCount()).To(Equal(0))
			})
		})
	})
})
