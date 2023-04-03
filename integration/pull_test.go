package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pull", func() {
	var (
		driverStore string
		layerStore  string
		imageURI    string
		chainIDs    []string
	)

	BeforeEach(func() {
		var err error
		driverStore, err = ioutil.TempDir("", "pull.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(driverStore, "layers")

		imagePath := filepath.Join(ociImagesDir, "regularfile")
		chainIDs = getLayerChainIdsFromOCIImage(imagePath)

		imageURI = pathToOCIURI(imagePath)
	})

	AfterEach(func() {
		destroyLayerStore(driverStore)
		Expect(os.RemoveAll(driverStore)).To(Succeed())
	})

	Context("provided an OCI image URI", func() {
		It("unpacks the layer to disk", func() {
			grootPull(driverStore, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(driverStore, imageURI)
			})

			It("does not overwrite the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(driverStore, imageURI)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})

			Context("when the image was unpacked without the size file", func() {
				BeforeEach(func() {
					grootPull(driverStore, imageURI)
					for _, chainID := range chainIDs {
						Expect(os.Remove(filepath.Join(layerStore, chainID, "size"))).To(Succeed())
					}
				})

				It("repulls the layers", func() {
					lastWriteTimes := []int64{}
					for _, chainID := range chainIDs {
						lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
					}

					grootPull(driverStore, imageURI)

					for i, chainID := range chainIDs {
						Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(BeNumerically(">", lastWriteTimes[i]))
						Expect(filepath.Join(layerStore, chainID, "size")).To(BeAnExistingFile())
					}
				})
			})
		})
	})

	Context("provided a Docker image URI", func() {
		BeforeEach(func() {
			imageURI = "docker:///cloudfoundry/groot-windows-test:regularfile"
		})

		It("unpacks the layer to disk", func() {
			grootPull(driverStore, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(driverStore, imageURI)
			})

			It("does not overwrite the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(driverStore, imageURI)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})

			Context("when the image was unpacked without the size file", func() {
				BeforeEach(func() {
					grootPull(driverStore, imageURI)
					for _, chainID := range chainIDs {
						Expect(os.Remove(filepath.Join(layerStore, chainID, "size"))).To(Succeed())
					}
				})

				It("repulls the layers", func() {
					lastWriteTimes := []int64{}
					for _, chainID := range chainIDs {
						lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
					}

					grootPull(driverStore, imageURI)

					for i, chainID := range chainIDs {
						Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(BeNumerically(">", lastWriteTimes[i]))
						Expect(filepath.Join(layerStore, chainID, "size")).To(BeAnExistingFile())
					}
				})
			})
		})
	})
})
