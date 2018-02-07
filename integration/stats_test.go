package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stats", func() {
	const (
		// note: this is the size of all the gzipped layers which is what libgroot
		// reports as the image size. We'll need to fix this whenever that bug is fixed
		baseImageSizeBytes = int64(81039739 + 42470724 + 70745)
		diskLimitSizeBytes = int64(160 * 1024 * 1024)
		fileSize           = int64(30 * 1024 * 1024)
	)

	var (
		driverStore    string
		volumeMountDir string
		ociImageDir    string
		imageURI       string
		bundleID       string
	)

	BeforeEach(func() {
		var err error
		driverStore, err = ioutil.TempDir("", "stats.store")
		Expect(err).ToNot(HaveOccurred())

		volumeMountDir, err = ioutil.TempDir("", "mounted-volume")
		Expect(err).ToNot(HaveOccurred())

		ociImageDir, err = ioutil.TempDir("", "oci-image")
		Expect(err).ToNot(HaveOccurred())

		ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
		Expect(extractTarGz(ociImageTgz, ociImageDir)).To(Succeed())
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

	Context("a disk limit is set", func() {
		BeforeEach(func() {
			outputSpec := grootCreate(driverStore, imageURI, bundleID, "--disk-limit-size-bytes", strconv.FormatInt(diskLimitSizeBytes, 10))
			mountVolume(outputSpec.Root.Path, volumeMountDir)
		})

		It("reports the image stats", func() {
			volumeStats := grootStats(driverStore, bundleID)
			Expect(volumeStats.DiskUsage.TotalBytesUsed).To(BeNumerically("~", baseImageSizeBytes, 7*1024))
			Expect(volumeStats.DiskUsage.ExclusiveBytesUsed).To(BeNumerically("~", 0, 7*1024))
		})

		Context("a large file is written", func() {
			BeforeEach(func() {
				largeFilePath := filepath.Join(volumeMountDir, "file.txt")
				Expect(exec.Command("fsutil", "file", "createnew", largeFilePath, strconv.FormatInt(fileSize, 10)).Run()).To(Succeed())
			})

			It("includes the file in disk usage", func() {
				volumeStats := grootStats(driverStore, bundleID)
				Expect(volumeStats.DiskUsage.TotalBytesUsed).To(BeNumerically("~", baseImageSizeBytes+fileSize, 7*1024))
				Expect(volumeStats.DiskUsage.ExclusiveBytesUsed).To(BeNumerically("~", fileSize, 7*1024))
			})
		})
	})

	Context("no disk limit is set", func() {
		BeforeEach(func() {
			outputSpec := grootCreate(driverStore, imageURI, bundleID)
			mountVolume(outputSpec.Root.Path, volumeMountDir)

			largeFilePath := filepath.Join(volumeMountDir, "file.txt")
			Expect(exec.Command("fsutil", "file", "createnew", largeFilePath, strconv.FormatInt(fileSize, 10)).Run()).To(Succeed())
		})

		It("returns just the base image size", func() {
			volumeStats := grootStats(driverStore, bundleID)
			Expect(volumeStats.DiskUsage.TotalBytesUsed).To(BeNumerically("~", baseImageSizeBytes, 7*1024))
			Expect(volumeStats.DiskUsage.ExclusiveBytesUsed).To(BeNumerically("~", 0, 7*1024))
		})
	})

	Context("the volume with the given bundle ID does not exist", func() {
		It("errors", func() {
			statsCmd := exec.Command(grootBin, "--driver-store", driverStore, "stats", bundleID)
			stdout, _, err := execute(statsCmd)
			Expect(err).To(HaveOccurred())
			Expect(stdout.String()).To(ContainSubstring(fmt.Sprintf("could not get volume path for bundle ID: %s", bundleID)))
		})
	})
})
