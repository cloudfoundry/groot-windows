package image_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/windows2016fs/image"
	"code.cloudfoundry.org/windows2016fs/image/imagefakes"
	"code.cloudfoundry.org/windows2016fs/layer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Image", func() {
	Describe("Extract", func() {
		var (
			srcDir   string
			destDir  string
			tempDir  string
			manifest v1.Manifest
			config   v1.Image
			lm       *imagefakes.FakeLayerManager
			mr       *imagefakes.FakeMetadataReader
			im       *image.Manager
		)

		const (
			layer1  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			layer2  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			layer3  = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			diffID1 = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
			diffID2 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			diffID3 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		)

		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "windows2016fs.image")
			Expect(err).NotTo(HaveOccurred())

			srcDir = filepath.Join(tempDir, "src")
			destDir = filepath.Join(tempDir, "dest")

			layers := []v1.Descriptor{
				{Digest: digest.NewDigestFromEncoded("sha256", layer1)},
				{Digest: digest.NewDigestFromEncoded("sha256", layer2)},
				{Digest: digest.NewDigestFromEncoded("sha256", layer3)},
			}

			manifest = v1.Manifest{
				Layers: layers,
			}

			config = v1.Image{
				RootFS: v1.RootFS{
					DiffIDs: []digest.Digest{
						digest.NewDigestFromEncoded("sha256", diffID1),
						digest.NewDigestFromEncoded("sha256", diffID2),
						digest.NewDigestFromEncoded("sha256", diffID3),
					},
				},
			}

			mr = &imagefakes.FakeMetadataReader{}
			lm = &imagefakes.FakeLayerManager{}
			mr.ReadReturnsOnCall(0, manifest, config, nil)
			im = image.NewManager(srcDir, mr, lm, ioutil.Discard)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		It("extracts all the layers, returning the top layer diffID", func() {
			topLayerId, err := im.Extract()
			Expect(err).NotTo(HaveOccurred())
			Expect(topLayerId).To(Equal(diffID3))

			Expect(lm.DeleteCallCount()).To(Equal(0))

			Expect(lm.ExtractCallCount()).To(Equal(3))
			tgz, id, parentIds := lm.ExtractArgsForCall(0)
			Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer1)))
			Expect(id).To(Equal(diffID1))
			Expect(parentIds).To(Equal([]string{}))

			tgz, id, parentIds = lm.ExtractArgsForCall(1)
			Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer2)))
			Expect(id).To(Equal(diffID2))
			Expect(parentIds).To(Equal([]string{diffID1}))

			tgz, id, parentIds = lm.ExtractArgsForCall(2)
			Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer3)))
			Expect(id).To(Equal(diffID3))
			Expect(parentIds).To(Equal([]string{diffID2, diffID1}))
		})

		Context("a layer has already been extracted", func() {
			BeforeEach(func() {
				lm.StateReturnsOnCall(1, layer.Valid, nil)
			})

			It("does not re-extract the existing layer", func() {
				_, err := im.Extract()
				Expect(err).NotTo(HaveOccurred())
				Expect(lm.DeleteCallCount()).To(Equal(0))

				Expect(lm.ExtractCallCount()).To(Equal(2))
				tgz, id, parentIds := lm.ExtractArgsForCall(0)
				Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer1)))
				Expect(id).To(Equal(diffID1))
				Expect(parentIds).To(Equal([]string{}))

				tgz, id, parentIds = lm.ExtractArgsForCall(1)
				Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer3)))
				Expect(id).To(Equal(diffID3))
				Expect(parentIds).To(Equal([]string{diffID2, diffID1}))
			})
		})

		Context("there is an invalid layer", func() {
			BeforeEach(func() {
				lm.StateReturnsOnCall(1, layer.Incomplete, nil)
			})

			It("deletes the incomplete layer and re-extracts", func() {
				_, err := im.Extract()
				Expect(err).NotTo(HaveOccurred())

				Expect(lm.DeleteCallCount()).To(Equal(1))
				Expect(lm.DeleteArgsForCall(0)).To(Equal(diffID2))

				Expect(lm.ExtractCallCount()).To(Equal(3))
				tgz, id, parentIds := lm.ExtractArgsForCall(0)
				Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer1)))
				Expect(id).To(Equal(diffID1))
				Expect(parentIds).To(Equal([]string{}))

				tgz, id, parentIds = lm.ExtractArgsForCall(1)
				Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer2)))
				Expect(id).To(Equal(diffID2))
				Expect(parentIds).To(Equal([]string{diffID1}))

				tgz, id, parentIds = lm.ExtractArgsForCall(2)
				Expect(tgz).To(Equal(filepath.Join(srcDir, "blobs", "sha256", layer3)))
				Expect(id).To(Equal(diffID3))
				Expect(parentIds).To(Equal([]string{diffID2, diffID1}))
			})
		})

		Context("provided an invalid content digest", func() {
			BeforeEach(func() {
				manifest = v1.Manifest{
					Layers: []v1.Descriptor{
						{Digest: digest.Digest("hello"), MediaType: "something.tar.gzip"},
					},
				}
				mr.ReadReturnsOnCall(0, manifest, config, nil)
			})

			It("returns an error", func() {
				_, err := im.Extract()
				Expect(err).To(MatchError(digest.ErrDigestInvalidFormat))
			})
		})

		Context("reading image metadata fails", func() {
			BeforeEach(func() {
				mr.ReadReturnsOnCall(0, v1.Manifest{}, v1.Image{}, errors.New("couldn't read metadata"))
			})

			It("returns an error", func() {
				_, err := im.Extract()
				Expect(err).To(MatchError("couldn't read metadata"))
			})
		})
	})
})
