package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pull", func() {
	var (
		layerStore  string
		ociImageDir string
		imageURI    string
		chainIDs    []string
	)

	BeforeEach(func() {
		var err error
		layerStore, err = ioutil.TempDir("", "layer-store")
		Expect(err).ToNot(HaveOccurred())

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
		Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())

		chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)

		imageURI = pathToOCIURI(ociImageDir)
	})

	AfterEach(func() {
		destroyLayerStore(layerStore)
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
	})

	Context("provided an OCI image URI", func() {
		It("unpacks the layer to disk", func() {
			grootPull(layerStore, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(layerStore, imageURI)
			})

			It("creates a volume without updating the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(layerStore, imageURI)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})
		})
	})

	Context("provided a Docker image URI", func() {
		BeforeEach(func() {
			imageURI = "docker:///cloudfoundry/groot-windows-test:regularfile"
		})

		It("unpacks the layer to disk", func() {
			grootPull(layerStore, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(layerStore, imageURI)
			})

			It("creates a volume without updating the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(layerStore, imageURI)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})
		})
	})
})
