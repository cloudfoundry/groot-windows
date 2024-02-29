package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
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
		driverStore, err = os.MkdirTemp("", "delete.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(driverStore, "layers")
		volumeStore = filepath.Join(driverStore, "volumes")

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

	Context("the volume with the given bundle ID exists", func() {
		BeforeEach(func() {
			grootCreate(driverStore, imageURI, bundleID)
		})

		It("deletes the volume", func() {
			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeTrue())

			grootDelete(driverStore, bundleID)

			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
			Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
		})

		It("does not delete any of the layer directories", func() {
			grootDelete(driverStore, bundleID)
			for _, dir := range parentLayerPaths {
				Expect(hcsshim.LayerExists(hcsshim.DriverInfo{HomeDir: layerStore, Flavour: 1}, filepath.Base(dir))).To(BeTrue())
				Expect(dir).To(BeADirectory())
			}
		})
	})

	Context("the driver store is a Unix-style path", func() {
		var unixStyleDriverStore string

		BeforeEach(func() {
			unixStyleDriverStore = strings.Replace(strings.TrimPrefix(driverStore, filepath.VolumeName(driverStore)), "\\", "/", -1)
			grootCreate(unixStyleDriverStore, imageURI, bundleID)
		})

		It("deletes the volume", func() {
			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeTrue())

			grootDelete(unixStyleDriverStore, bundleID)

			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
			Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
		})
	})

	Context("the volume with the given bundle ID does not exist", func() {
		It("returns success, writing a message to the log", func() {
			deleteCmd := exec.Command(grootBin, "--driver-store", driverStore, "delete", bundleID)
			_, stderr, err := execute(deleteCmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(stderr.String()).To(ContainSubstring("volume-not-found"))
			Expect(stderr.String()).To(ContainSubstring(fmt.Sprintf(`"bundleID":"%s"`, bundleID)))
		})
	})

	Context("the volume with the given bundle ID has been partially created", func() {
		BeforeEach(func() {
			grootPull(driverStore, imageURI)
		})

		Context("the volume layer has been created but is not activated", func() {
			BeforeEach(func() {
				Expect(hcsshim.CreateSandboxLayer(driverInfo, bundleID, parentLayerPaths[0], parentLayerPaths)).To(Succeed())
			})

			It("deletes the volume", func() {
				grootDelete(driverStore, bundleID)

				Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
				Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
			})
		})

		Context("the volume layer has been created and activated but is not prepared", func() {
			BeforeEach(func() {
				Expect(hcsshim.CreateSandboxLayer(driverInfo, bundleID, parentLayerPaths[0], parentLayerPaths)).To(Succeed())
				Expect(hcsshim.ActivateLayer(driverInfo, bundleID)).To(Succeed())
			})

			It("deletes the volume", func() {
				grootDelete(driverStore, bundleID)

				Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
				Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
			})
		})
	})
})
