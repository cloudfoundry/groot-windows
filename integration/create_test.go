package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Create", func() {
	var (
		driverStore    string
		volumeStore    string
		layerStore     string
		volumeMountDir string
		ociImageDir    string
		imageURI       string
		bundleID       string
		chainIDs       []string
	)

	BeforeEach(func() {
		var err error
		driverStore, err = ioutil.TempDir("", "create.store")
		Expect(err).ToNot(HaveOccurred())
		layerStore = filepath.Join(driverStore, "layers")
		volumeStore = filepath.Join(driverStore, "volumes")

		volumeMountDir, err = ioutil.TempDir("", "mounted-volume")
		Expect(err).ToNot(HaveOccurred())

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		imageURI = pathToOCIURI(ociImageDir)

		bundleID = randomBundleID()
	})

	AfterEach(func() {
		unmountVolume(volumeMountDir)
		destroyVolumeStore(driverStore)
		destroyLayerStore(driverStore)
		Expect(os.RemoveAll(volumeMountDir)).To(Succeed())
		Expect(os.RemoveAll(ociImageDir)).To(Succeed())
		Expect(os.RemoveAll(driverStore)).To(Succeed())
	})

	Context("provided an OCI image URI", func() {
		Context("when the image contains a layer with a regular file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
				chainIDs = getLayerChainIdsFromOCIImage(ociImageDir)
			})

			It("unpacks the layer to disk", func() {
				grootCreate(driverStore, imageURI, bundleID)

				for _, chainID := range chainIDs {
					Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
				}
			})

			It("returns a runtime spec on stdout", func() {
				outputSpec := grootCreate(driverStore, imageURI, bundleID)

				Expect(outputSpec.Root.Path).ToNot(BeEmpty())
				Expect(outputSpec.Version).To(Equal(specs.Version))

				layerFolders := []string{}
				for _, chainID := range chainIDs {
					layerFolders = append([]string{filepath.Join(layerStore, chainID)}, layerFolders...)
				}
				Expect(outputSpec.Windows.LayerFolders).To(Equal(layerFolders))
			})

			It("the resulting volume contains the correct files", func() {
				outputSpec := grootCreate(driverStore, imageURI, bundleID)
				mountVolume(outputSpec.Root.Path, volumeMountDir)

				knownFilePath := filepath.Join(volumeMountDir, "temp", "test", "hello")
				Expect(knownFilePath).To(BeAnExistingFile())
			})

			It("creates the volume vhdx in the proper location", func() {
				grootCreate(driverStore, imageURI, bundleID)

				vhdxPath := filepath.Join(volumeStore, bundleID, "Sandbox.vhdx")
				Expect(vhdxPath).To(BeAnExistingFile())
			})
		})

		Context("when the image contains a layer with a whiteout file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-whiteout.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
			})

			It("the resulting volume has the correct files removed", func() {
				outputSpec := grootCreate(driverStore, imageURI, bundleID)
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
				outputSpec := grootCreate(driverStore, imageURI, bundleID)
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

				grootCreate(driverStore, imageURI, bundleID)
			})

			It("creates a volume without updating the unpacked layers", func() {
				newBundleID := randomBundleID()

				lastWriteTimes := []int64{}
				for _, chainID := range chainIDs {
					lastWriteTimes = append(lastWriteTimes, getLastWriteTime(filepath.Join(layerStore, chainID)))
				}

				grootCreate(driverStore, imageURI, newBundleID)

				for i, chainID := range chainIDs {
					Expect(getLastWriteTime(filepath.Join(layerStore, chainID))).To(Equal(lastWriteTimes[i]))
				}
			})
		})

		Context("when the requested bundle ID is already in use", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())

				grootCreate(driverStore, imageURI, bundleID)
			})

			It("returns a helpful error", func() {
				createCmd := exec.Command(grootBin, "--driver-store", driverStore, "create", imageURI, bundleID)
				stdOut, _, err := execute(createCmd)
				Expect(err).To(HaveOccurred())
				Expect(stdOut.String()).To(ContainSubstring(fmt.Sprintf("layer already exists: %s", bundleID)))
			})
		})

		Context("when provided a disk limit", func() {
			var (
				// note: this is the size of all the gzipped layers which is what libgroot
				// reports as the image size. We'll need to fix this whenever that bug is fixed
				baseImageSizeBytes = 81039739 + 42470724 + 70745
				diskLimitSizeBytes = baseImageSizeBytes + 50*1024*1024
			)

			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
			})

			Context("the disk limit is greater than 0", func() {
				FIt("includes the base image size in the limit", func() {
					outputSpec := grootCreate(driverStore, imageURI, bundleID, "--disk-limit-size-bytes", strconv.Itoa(diskLimitSizeBytes))
					mountVolume(outputSpec.Root.Path, volumeMountDir)

					output, err := exec.Command("dirquota", "quota", "list", fmt.Sprintf("/Path:%s", volumeMountDir)).CombinedOutput()
					Expect(err).NotTo(HaveOccurred(), string(output))
					Expect(string(output)).To(ContainSubstring("don't know, maybe a regex"))
				})

				It("doesn't allow files larger than the limit to be created", func() {
				})

				It("allows files at the limit to be created", func() {
				})
			})

			Context("--exclude-image-from-quota is passed", func() {
				It("includes the base image size in the limit", func() {
				})

				It("doesn't allow files larger than the limit to be created", func() {
				})

				It("allows files at the limit to be created", func() {
				})

				Context("the image size is larger than the disk limit", func() {
					It("errors", func() {
					})
				})
			})

			Context("the disk limit is equal to 0", func() {
				It("does not set a limit", func() {
				})
			})

			Context("the disk limit is less then 0", func() {
				It("errors", func() {
				})
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
			grootCreate(driverStore, imageURI, bundleID)

			for _, chainID := range chainIDs {
				Expect(filepath.Join(layerStore, chainID, "Files")).To(BeADirectory())
			}
		})

		It("returns a runtime spec on stdout", func() {
			outputSpec := grootCreate(driverStore, imageURI, bundleID)

			Expect(outputSpec.Root.Path).ToNot(BeEmpty())
			Expect(outputSpec.Version).To(Equal(specs.Version))

			layerFolders := []string{}
			for _, chainID := range chainIDs {
				layerFolders = append([]string{filepath.Join(layerStore, chainID)}, layerFolders...)
			}
			Expect(outputSpec.Windows.LayerFolders).To(Equal(layerFolders))
		})

		It("the resulting volume contains the correct files", func() {
			outputSpec := grootCreate(driverStore, imageURI, bundleID)
			mountVolume(outputSpec.Root.Path, volumeMountDir)

			knownFilePath := filepath.Join(volumeMountDir, "temp", "test", "hello")
			Expect(knownFilePath).To(BeAnExistingFile())
		})
	})
})
