package filefetcher_test

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/groot/fetcher/filefetcher"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("File Fetcher", func() {
	var (
		fetcher *filefetcher.FileFetcher

		sourceImagePath string
		imagePath       string
		logger          *lagertest.TestLogger
		imageURL        *url.URL
	)

	BeforeEach(func() {
		fetcher = filefetcher.NewFileFetcher()

		var err error
		sourceImagePath, err = ioutil.TempDir("", "image")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(path.Join(sourceImagePath, "a_file"), []byte("hello-world"), 0600)).To(Succeed())
		logger = lagertest.NewTestLogger("file-fetcher")
	})

	JustBeforeEach(func() {
		imagePath = filepath.Join(sourceImagePath, "a_file")
		var err error
		imageURL, err = url.Parse(imagePath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(imagePath)).To(Succeed())
		Expect(os.RemoveAll(sourceImagePath)).To(Succeed())
	})

	Describe("StreamBlob", func() {
		It("returns the contents of the source file", func() {
			stream, _, err := fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})
			Expect(err).ToNot(HaveOccurred())
			defer stream.Close()

			streamContents, err := ioutil.ReadAll(stream)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(streamContents)).To(Equal("hello-world"))
		})

		Context("when the source is a directory", func() {
			It("returns an error message", func() {
				tempDir, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				imageURL, _ := url.Parse(tempDir)
				_, _, err = fetcher.StreamBlob(logger, imageURL, imagepuller.LayerInfo{})
				Expect(err).To(MatchError(ContainSubstring("invalid base image: directory provided instead of a tar file")))
			})
		})

		Context("when the source does not exist", func() {
			It("returns an error", func() {
				nonExistentImageURL, _ := url.Parse("/nothing/here")

				_, _, err := fetcher.StreamBlob(logger, nonExistentImageURL, imagepuller.LayerInfo{})
				Expect(err).To(MatchError(ContainSubstring("local image not found in `/nothing/here`")))
			})
		})
	})

	Describe("LayersDigest", func() {
		var imageInfo imagepuller.ImageInfo

		JustBeforeEach(func() {
			var err error
			imageInfo, err = fetcher.ImageInfo(logger, imageURL)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the correct image", func() {
			layers := imageInfo.LayerInfos

			Expect(len(layers)).To(Equal(1))
			Expect(strings.EqualFold(layers[0].BlobID, imagePath)).To(BeTrue())
			Expect(layers[0].ChainID).NotTo(BeEmpty())
			Expect(layers[0].ParentChainID).To(BeEmpty())

			Expect(imageInfo.Config).To(Equal(v1.Image{}))
		})

		Context("when image timestamp changes", func() {
			JustBeforeEach(func() {
				Expect(os.Chtimes(imagePath, time.Now().Add(time.Hour), time.Now().Add(time.Hour))).To(Succeed())
			})

			It("generates another chain id", func() {
				newImageInfo, err := fetcher.ImageInfo(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())
				Expect(imageInfo.LayerInfos[0].ChainID).NotTo(Equal(newImageInfo.LayerInfos[0].ChainID))
			})
		})

		Context("when the image doesn't exist", func() {
			JustBeforeEach(func() {
				var err error
				imageURL, err = url.Parse("/not-here")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, err := fetcher.ImageInfo(logger, imageURL)
				Expect(err).To(MatchError(ContainSubstring("fetching image timestamp")))
			})
		})
	})
})

type tarEntry struct {
	header   *tar.Header
	contents []byte
}

func streamTar(r *tar.Reader) []tarEntry {
	l := []tarEntry{}
	for {
		header, err := r.Next()
		if err != nil {
			Expect(err).To(Equal(io.EOF))
			return l
		}

		contents := make([]byte, header.Size)
		_, _ = r.Read(contents)
		l = append(l, tarEntry{
			header:   header,
			contents: contents,
		})
	}
}
