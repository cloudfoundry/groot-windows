package integration_test

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/hydrator/imagefetcher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	grootBin string

	imageTags = []string{
		"regularfile",
		"whiteout",
		"link",
		"servercore",
	}
	ociImagesDir   string
	keepDir        bool
	baseImageBytes int64
)

func TestGrootWindows(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Millisecond * 200)
	RunSpecs(t, "GrootWindows Suite")
}

func setBaseImageBytes() {
	if baseImageBytes != 0 {
		return
	}
	// baseImageBytes should be set dynamically because we don't control the size of the base image from Microsoft
	// Since the method for determining the image size is equivalent to creating it, just do that.
	driverStore, err := os.MkdirTemp("", "base.stats.store")
	Expect(err).ToNot(HaveOccurred())

	volumeMountDir, err := os.MkdirTemp("", "base.mounted-volume")
	Expect(err).ToNot(HaveOccurred())

	imageURI := pathToOCIURI(filepath.Join(ociImagesDir, "regularfile"))
	//imageURI := "mcr.microsoft.com/nanoserver:1809"

	bundleID := randomBundleID()

	outputSpec := grootCreate(driverStore, imageURI, bundleID)
	mountVolume(outputSpec.Root.Path, volumeMountDir)
	volumeStats := grootStats(driverStore, bundleID)
	baseImageBytes = volumeStats.DiskUsage.TotalBytesUsed

	unmountVolume(volumeMountDir)
	destroyVolumeStore(driverStore)
	destroyLayerStore(driverStore)
	Expect(os.RemoveAll(volumeMountDir)).To(Succeed())
	Expect(os.RemoveAll(driverStore)).To(Succeed())
}

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	grootBin, err = gexec.Build("code.cloudfoundry.org/groot-windows")
	Expect(err).ToNot(HaveOccurred())

	grootDir := filepath.Dir(grootBin)

	o, err := exec.Command("gcc.exe", "-c", "..\\volume\\quota\\quota.c", "-o", filepath.Join(grootDir, "quota.o")).CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(o))

	o, err = exec.Command("gcc.exe",
		"-shared",
		"-o", filepath.Join(grootDir, "quota.dll"),
		filepath.Join(grootDir, "quota.o"),
		"-lole32", "-loleaut32").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(o))

	ociImagesDir, keepDir = os.LookupEnv("GROOT_WINDOWS_IMAGES")

	//setBaseImageBytes()

	if !keepDir {
		ociImagesDir, err = os.MkdirTemp("", "groot-windows-test-images")
		Expect(err).ToNot(HaveOccurred())
	}

	for _, tag := range imageTags {
		_, err := os.Stat(filepath.Join(ociImagesDir, tag))
		if err != nil && os.IsNotExist(err) {
			logger := log.New(os.Stdout, "", 0)
			Expect(imagefetcher.New(logger, filepath.Join(ociImagesDir, tag), "cloudfoundry/groot-windows-test", tag, "", true).Run()).To(Succeed())
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
	}

	testData := make(map[string]string)
	testData["groot_bin"] = grootBin
	testData["oci_image_dir"] = ociImagesDir
	json, err := json.Marshal(testData)
	Expect(err).NotTo(HaveOccurred())

	return json
}, func(jsonBytes []byte) {
	testData := make(map[string]string)
	Expect(json.Unmarshal(jsonBytes, &testData)).To(Succeed())

	grootBin = testData["groot_bin"]
	ociImagesDir = testData["oci_image_dir"]
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	if !keepDir {
		Expect(os.RemoveAll(ociImagesDir)).To(Succeed())
	}
	gexec.CleanupBuildArtifacts()
})
