package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		layerStore  string
		volumeStore string
		ociImageDir string
		imageURI    string
		bundleID    string
		driverInfo  hcsshim.DriverInfo
	)

	BeforeEach(func() {
		var err error
		layerStore, err = ioutil.TempDir("", "layer-store")
		Expect(err).ToNot(HaveOccurred())

		volumeStore, err = ioutil.TempDir("", "volume-store")
		Expect(err).ToNot(HaveOccurred())

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
		Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
		imageURI = pathToOCIURI(ociImageDir)

		bundleID = randomBundleID()
		driverInfo = hcsshim.DriverInfo{HomeDir: volumeStore, Flavour: 1}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(volumeStore)).To(Succeed())
		destroyLayerStore(layerStore)
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
	})

	Context("the volume with the given bundle ID exists", func() {
		BeforeEach(func() {
			grootCreate(layerStore, volumeStore, imageURI, bundleID)
		})

		It("deletes the volume", func() {
			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeTrue())

			grootDelete(volumeStore, bundleID)

			Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
			Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
		})
	})

	Context("the volume with the given bundle ID does not exist", func() {
		It("returns success, writing a message to the log", func() {
			deleteCmd := exec.Command(grootBin, "delete", bundleID)
			deleteCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_VOLUME_STORE=%s", volumeStore))
			_, stderr, err := execute(deleteCmd)
			Expect(err).ToNot(HaveOccurred())

			Expect(stderr.String()).To(ContainSubstring("volume-not-found"))
			Expect(stderr.String()).To(ContainSubstring(fmt.Sprintf(`"bundleID":"%s"`, bundleID)))
		})
	})

	Context("the volume with the given bundle ID has been partially created", func() {
		var parentLayerPaths []string

		BeforeEach(func() {
			Skip("these tests require `groot pull`")
			chainIDs := getLayerChainIdsFromOCIImage(ociImageDir)
			for _, id := range chainIDs {
				parentLayerPaths = append(parentLayerPaths, filepath.Join(layerStore, id))
			}
		})

		Context("the volume layer has been created but is not activated", func() {
			BeforeEach(func() {
				Expect(hcsshim.CreateSandboxLayer(driverInfo, bundleID, parentLayerPaths[0], parentLayerPaths)).To(Succeed())
			})

			It("deletes the volume", func() {
				grootDelete(volumeStore, bundleID)

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
				grootDelete(volumeStore, bundleID)

				Expect(hcsshim.LayerExists(driverInfo, bundleID)).To(BeFalse())
				Expect(filepath.Join(volumeStore, bundleID)).NotTo(BeADirectory())
			})
		})
	})
})
