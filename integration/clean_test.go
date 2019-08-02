package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Clean", func() {
	var (
		driverStore      string
		layerStore       string
		volumeStore      string
		imageURI         string
		bundleID         string
		driverInfo       hcsshim.DriverInfo
		parentLayerPaths []string
	)

	BeforeEach(func() {
		var err error
		driverStore, err = ioutil.TempDir("", "clean.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(driverStore, "layers")
		volumeStore = filepath.Join(driverStore, "volumes")

		// this is just a test rootfs stored on docker hub
		imagePath := filepath.Join(ociImagesDir, "regularfile")
		imageURI = pathToOCIURI(imagePath)

		bundleID = randomBundleID()
		driverInfo = hcsshim.DriverInfo{HomeDir: volumeStore, Flavour: 1}

		parentLayerPaths = []string{}
		chainIDs := getLayerChainIdsFromOCIImage(imagePath)
		for _, id := range chainIDs {
			parentLayerPaths = append([]string{filepath.Join(layerStore, id)}, parentLayerPaths...)
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(volumeStore)).To(Succeed())
		destroyLayerStore(driverStore)
		Expect(os.RemoveAll(driverStore)).To(Succeed())
	})

	Context("No volume exists that refers the layers", func() {
		BeforeEach(func() {
			grootCreate(driverStore, imageURI, bundleID)
			grootDelete(driverStore, bundleID)
		})

		It("Clean cmd removes all layers", func() {
			// Expect len(contents) of layerStore > 0

			// Run clean

			// Expect len(contents) of layerStore = 0
		})

	})

	Context("There exists a volume that refers the layers", func() {
		BeforeEach(func() {
			grootCreate(driverStore, imageURI, bundleID)
		})

		AfterEach(func() {
			grootDelete(driverStore, bundleID)
		})
	})

})
