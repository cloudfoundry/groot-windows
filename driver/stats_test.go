package driver_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stats", func() {
	var (
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		limiterFake           *fakes.Limiter
		logger                *lagertest.TestLogger
		bundleID              string
		storeDir              string
		metadataFile          string
		baseImageSize         int64
		quotaUsed             int64
		volumeGUID            string
	)

	BeforeEach(func() {
		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}
		limiterFake = &fakes.Limiter{}

		var err error
		storeDir, err = ioutil.TempDir("", "stats-store")
		Expect(err).NotTo(HaveOccurred())

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake, limiterFake)
		d.Store = storeDir

		logger = lagertest.NewTestLogger("driver-stats-test")
		bundleID = "some-bundle-id"

		bundleVolumeDir := filepath.Join(storeDir, "volumes", bundleID)
		Expect(os.MkdirAll(bundleVolumeDir, 0755)).To(Succeed())

		baseImageSize = 12345
		metadataFile = filepath.Join(bundleVolumeDir, "metadata.json")
		data, err := json.Marshal(groot.ImageMetadata{Size: baseImageSize})
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(metadataFile, data, 0644)).To(Succeed())

		quotaUsed = 6789
		limiterFake.GetQuotaUsedReturnsOnCall(0, uint64(quotaUsed), nil)

		volumeGUID = "some-volume-guid"
		hcsClientFake.GetLayerMountPathReturnsOnCall(0, volumeGUID, nil)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	It("returns the appropriate stats", func() {
		stats, err := d.Stats(logger, bundleID)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.DiskUsage.TotalBytesUsed).To(Equal(baseImageSize + quotaUsed))
		Expect(stats.DiskUsage.ExclusiveBytesUsed).To(Equal(quotaUsed))

		Expect(hcsClientFake.GetLayerMountPathCallCount()).To(Equal(1))
		di, id := hcsClientFake.GetLayerMountPathArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: d.VolumeStore(), Flavour: 1}))
		Expect(id).To(Equal(bundleID))

		Expect(limiterFake.GetQuotaUsedCallCount()).To(Equal(1))
		Expect(limiterFake.GetQuotaUsedArgsForCall(0)).To(Equal(volumeGUID))
	})

	Context("metadata.json file can't be read", func() {
		BeforeEach(func() {
			d.Store = "not-exist"
		})

		It("errors", func() {
			_, err := d.Stats(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The system cannot find the path specified"))
		})
	})

	Context("metadata.json file contains bad data", func() {
		BeforeEach(func() {
			Expect(ioutil.WriteFile(metadataFile, []byte("not json"), 0644)).To(Succeed())
		})

		It("errors", func() {
			_, err := d.Stats(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("couldn't parse metadata.json"))
		})
	})

	Context("GetLayerMountPath returns an error", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", errors.New("layer mount path failed"))
		})

		It("returns the error", func() {
			_, err := d.Stats(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("layer mount path failed"))
		})
	})

	Context("GetLayerMountPath returns an empty string", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", nil)
		})

		It("returns an error", func() {
			_, err := d.Stats(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(&driver.MissingVolumePathError{Id: bundleID}))
		})
	})

	Context("GetQuotaUsed returns an error", func() {
		BeforeEach(func() {
			limiterFake.GetQuotaUsedReturnsOnCall(0, 0, errors.New("couldn't get quota"))
		})

		It("returns the error", func() {
			_, err := d.Stats(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("couldn't get quota"))
		})
	})
})
