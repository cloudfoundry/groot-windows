package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Create", func() {
	var (
		layerStore     string
		volumeStore    string
		volumeMountDir string
		ociImageDir    string
		imageURI       string
		bundleID       string
		chainIDs       []string
	)

	BeforeEach(func() {
		var err error
		layerStore, err = ioutil.TempDir("", "layer-store")
		Expect(err).ToNot(HaveOccurred())

		volumeStore, err = ioutil.TempDir("", "volume-store")
		Expect(err).ToNot(HaveOccurred())

		volumeMountDir, err = ioutil.TempDir("", "mounted-volume")
		Expect(err).ToNot(HaveOccurred())

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		imageURI = pathToOCIURI(ociImageDir)

		bundleID = randomBundleID()
	})

	AfterEach(func() {
		unmountVolume(volumeMountDir)
		destroyVolumeStore(volumeStore)
		destroyLayerStore(layerStore)
		Expect(os.RemoveAll(volumeMountDir)).To(Succeed())
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
	})

	Context("provided an OCI image URI", func() {
		Context("when the image contains a layer with a regular file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
				chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)
			})

			It("unpacks the layer to disk", func() {
				grootCreate(layerStore, volumeStore, imageURI, bundleID)

				for _, chainID := range chainIDs {
					Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
				}
			})

			It("returns a runtime spec on stdout", func() {
				outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)

				Expect(outputSpec.Root.Path).ToNot(BeEmpty())
				Expect(outputSpec.Version).To(Equal(specs.Version))

				layerFolders := []string{}
				for _, chainID := range chainIDs {
					layerFolders = append([]string{filepath.Join(layerStore, chainID)}, layerFolders...)
				}
				Expect(outputSpec.Windows.LayerFolders).To(Equal(layerFolders))
			})

			It("the resulting volume contains the correct files", func() {
				outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)
				mountVolume(outputSpec.Root.Path, volumeMountDir)

				knownFilePath := filepath.Join(volumeMountDir, "temp", "test", "hello")
				Expect(knownFilePath).To(BeAnExistingFile())
			})
		})

		Context("when the image contains a layer with a whiteout file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-whiteout.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
			})

			It("the resulting volume has the correct files removed", func() {
				outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)
				mountVolume(outputSpec.Root.Path, volumeMountDir)

				Expect(filepath.Join(volumeMountDir, "temp", "test", "hello2")).To(BeAnExistingFile())
				Expect(filepath.Join(volumeMountDir, "temp", "test", "hello")).ToNot(BeAnExistingFile())
			})
		})

		Context("when the image contains a layer with symlinks and hardlinks", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-link.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
				chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)
			})

			It("the resulting volume has the correct symlinks, hardlinks, and junctions", func() {
				outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)
				mountVolume(outputSpec.Root.Path, volumeMountDir)

				dest, err := os.Readlink(filepath.Join(volumeMountDir, "temp", "symlinkfile"))
				Expect(err).NotTo(HaveOccurred())
				Expect(dest).To(Equal("C:\\temp\\test\\hello"))

				data, err := ioutil.ReadFile(filepath.Join(volumeMountDir, "temp", "hardlinkfile"))
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.TrimSpace(string(data))).To(Equal("hello"))

				symlinkDirPath := filepath.Join(volumeMountDir, "temp", "symlinkdir")
				Expect(getReparseTag(symlinkDirPath)).To(Equal(uint32(syscall.IO_REPARSE_TAG_SYMLINK)), "not a symlink")
				Expect(getSymlinkDest(symlinkDirPath)).To(Equal("C:\\temp\\test"))
				Expect(getFileAttributes(symlinkDirPath)&syscall.FILE_ATTRIBUTE_DIRECTORY).To(Equal(uint32(syscall.FILE_ATTRIBUTE_DIRECTORY)), "not a directory")

				junctionDirPath := filepath.Join(volumeMountDir, "temp", "junctiondir")
				Expect(getReparseTag(junctionDirPath)).To(Equal(uint32(IO_REPARSE_TAG_MOUNT_POINT)), "not a junction point")
				Expect(getSymlinkDest(junctionDirPath)).To(Equal("C:\\temp\\test"))
				Expect(getFileAttributes(junctionDirPath)&syscall.FILE_ATTRIBUTE_DIRECTORY).To(Equal(uint32(syscall.FILE_ATTRIBUTE_DIRECTORY)), "not a directory")
			})
		})

		Context("when the image has already been unpacked", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
				chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)

				grootCreate(layerStore, volumeStore, imageURI, bundleID)
			})

			It("creates a volume without updating the unpacked layers", func() {
				newBundleID := randomBundleID()

				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootCreate(layerStore, volumeStore, imageURI, newBundleID)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})
		})

		Context("when the requested bundle ID is already in use", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())

				grootCreate(layerStore, volumeStore, imageURI, bundleID)
			})

			It("returns a helpful error", func() {
				createCmd := exec.Command(grootBin, "create", imageURI, bundleID)
				createCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_LAYER_STORE=%s", layerStore), fmt.Sprintf("GROOT_VOLUME_STORE=%s", volumeStore))
				stdOut, _, err := execute(createCmd)
				Expect(err).To(HaveOccurred())
				Expect(stdOut.String()).To(ContainSubstring(fmt.Sprintf("layer already exists: %s", bundleID)))
			})
		})
	})

	Context("provided a Docker image URI", func() {
		BeforeEach(func() {
			imageURI = "docker:///cloudfoundry/groot-windows-test:regularfile"

			ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
			Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
			chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)
		})

		It("unpacks the layer to disk", func() {
			grootCreate(layerStore, volumeStore, imageURI, bundleID)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		It("returns a runtime spec on stdout", func() {
			outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)

			Expect(outputSpec.Root.Path).ToNot(BeEmpty())
			Expect(outputSpec.Version).To(Equal(specs.Version))

			layerFolders := []string{}
			for _, chainID := range chainIDs {
				layerFolders = append([]string{filepath.Join(layerStore, chainID)}, layerFolders...)
			}
			Expect(outputSpec.Windows.LayerFolders).To(Equal(layerFolders))
		})

		It("the resulting volume contains the correct files", func() {
			outputSpec := grootCreate(layerStore, volumeStore, imageURI, bundleID)
			mountVolume(outputSpec.Root.Path, volumeMountDir)

			knownFilePath := filepath.Join(volumeMountDir, "temp", "test", "hello")
			Expect(knownFilePath).To(BeAnExistingFile())
		})
	})
})
