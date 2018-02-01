package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot-windows/driver"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		storeDir         string
		layerStore       string
		volumeStore      string
		ociImageDir      string
		imageURI         string
		bundleID         string
		driverInfo       hcsshim.DriverInfo
		parentLayerPaths []string
		confFile         string
	)

	BeforeEach(func() {
		var err error
		storeDir, err = ioutil.TempDir("", "delete.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(storeDir, driver.LayerDir)
		volumeStore = filepath.Join(storeDir, driver.VolumeDir)

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
		Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
		imageURI = pathToOCIURI(ociImageDir)

		bundleID = randomBundleID()
		driverInfo = hcsshim.DriverInfo{HomeDir: volumeStore, Flavour: 1}

		parentLayerPaths = []string{}
		chainIDs := getLayerChainIdsFromOCIImage(ociImageDir)
		for _, id := range chainIDs {
			parentLayerPaths = append([]string{filepath.Join(layerStore, id)}, parentLayerPaths...)
		}

		confFile = writeConfig(storeDir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(volumeStore)).To(Succeed())
		destroyLayerStore(confFile)
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
		Expect(os.RemoveAll(storeDir)).To(Succeed())
		Expect(os.RemoveAll(confFile)).To(Succeed())
	})

	Context("the volume with the given bundle ID exists", func() {
		BeforeEach(func() {
			grootCreate(confFile, imageURI, bundleID)
		})

		It("deletes the volume", func() {
			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeTrue())

			grootDelete(confFile, bundleID)

			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
			Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
		})

		It("does not delete any of the layer directories", func() {
			grootDelete(confFile, bundleID)
			for _, dir := range parentLayerPaths {
				Expect(hcsshim.LayerExists(hcsshim.DriverInfo{HomeDir: layerStore, Flavour: 1}, filepath.Base(dir))).To(BeTrue())
				Expect(dir).To(BeADirectory())
			}
		})
	})

	Context("the volume with the given bundle ID does not exist", func() {
		It("returns success, writing a message to the log", func() {
			deleteCmd := exec.Command(grootBin, "--config", confFile, "delete", bundleID)
			_, stderr, err := execute(deleteCmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(stderr.String()).To(ContainSubstring("volume-not-found"))
			Expect(stderr.String()).To(ContainSubstring(fmt.Sprintf(`"bundleID":"%s"`, bundleID)))
		})
	})

	Context("the volume with the given bundle ID has been partially created", func() {
		BeforeEach(func() {
			grootPull(confFile, imageURI)
		})

		Context("the volume layer has been created but is not activated", func() {
			BeforeEach(func() {
				Expect(hcsshim.CreateSandboxLayer(driverInfo, bundleID, parentLayerPaths[0], parentLayerPaths)).To(Succeed())
			})

			It("deletes the volume", func() {
				grootDelete(confFile, bundleID)

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
				grootDelete(confFile, bundleID)

				Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
				Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
			})
		})
	})
})
