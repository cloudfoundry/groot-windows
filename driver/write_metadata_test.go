package driver_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WriteMetadata", func() {
	var (
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		limiterFake           *fakes.Limiter
		logger                *lagertest.TestLogger
		bundleID              string
		storeDir              string
		volumeData            groot.ImageMetadata
	)

	BeforeEach(func() {
		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}
		limiterFake = &fakes.Limiter{}

		var err error
		storeDir, err = ioutil.TempDir("", "write-metadata-store")
		Expect(err).NotTo(HaveOccurred())

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake, limiterFake)
		d.Store = storeDir

		logger = lagertest.NewTestLogger("driver-write-metadata-test")
		bundleID = "some-bundle-id"

		volumeData = groot.ImageMetadata{Size: 4000}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	Context("the <bundle-id> directory exists", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(d.VolumeStore(), "some-bundle-id"), 0755)).To(Succeed())
		})

		It("writes the metadata to the <volume-dir>/<bundle-id> directory", func() {
			Expect(d.WriteMetadata(logger, bundleID, volumeData)).To(Succeed())

			contents, err := ioutil.ReadFile(filepath.Join(d.VolumeStore(), "some-bundle-id", "metadata.json"))
			Expect(err).NotTo(HaveOccurred())

			var data groot.ImageMetadata
			Expect(json.Unmarshal(contents, &data)).To(Succeed())
			Expect(data).To(Equal(volumeData))
		})
	})

	Context("the <bundle-id> directory does not exist", func() {
		It("returns a useful error", func() {
			err := d.WriteMetadata(logger, bundleID, volumeData)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("WriteMetadata failed"))
		})
	})
})
