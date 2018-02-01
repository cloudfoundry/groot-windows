package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot-windows/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pull", func() {
	var (
		storeDir    string
		layerStore  string
		ociImageDir string
		imageURI    string
		chainIDs    []string
	)

	BeforeEach(func() {
		var err error
		storeDir, err = ioutil.TempDir("", "pull.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(storeDir, driver.LayerDir)

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
		Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())

		chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)

		imageURI = pathToOCIURI(ociImageDir)
	})

	AfterEach(func() {
		destroyLayerStore(storeDir)
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	Context("provided an OCI image URI", func() {
		It("unpacks the layer to disk", func() {
			grootPull(storeDir, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(storeDir, imageURI)
			})

			It("creates a volume without updating the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(storeDir, imageURI)

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
			grootPull(storeDir, imageURI)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				grootPull(storeDir, imageURI)
			})

			It("creates a volume without updating the unpacked layers", func() {
				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootPull(storeDir, imageURI)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})
		})
	})
})
