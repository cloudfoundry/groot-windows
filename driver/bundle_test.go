package driver_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Bundle", func() {
	const (
		bundleID   = "some-bundle-id"
		volumeGUID = "some-volume-guid"
	)

	var (
		storeDir              string
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		limiterFake           *fakes.Limiter
		logger                *lagertest.TestLogger
		layerIDs              = []string{"oldest-layer", "middle-layer", "newest-layer"}
		bundleSpec            groot.BundleSpec
	)

	BeforeEach(func() {
		var err error

		storeDir, err = ioutil.TempDir("", "bundle-store")
		Expect(err).NotTo(HaveOccurred())

		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}
		limiterFake = &fakes.Limiter{}

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake, limiterFake)
		d.Store = storeDir

		logger = lagertest.NewTestLogger("driver-unpack-test")
		hcsClientFake.GetLayerMountPathReturnsOnCall(0, volumeGUID, nil)

		bundleSpec = groot.BundleSpec{BaseImageSize: 234}

		hcsClientFake.CreateLayerStub = func(di hcsshim.DriverInfo, id string, _ string, _ []string) error {
			Expect(os.MkdirAll(filepath.Join(di.HomeDir, id), 0755)).To(Succeed())
			return nil
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	It("returns a valid runtime spec", func() {
		spec, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(spec.Version).To(Equal(specs.Version))
		Expect(spec.Root.Path).To(Equal(volumeGUID))

		expectedLayerDirs := []string{
			filepath.Join(d.LayerStore(), "newest-layer"),
			filepath.Join(d.LayerStore(), "middle-layer"),
			filepath.Join(d.LayerStore(), "oldest-layer"),
		}
		Expect(spec.Windows.LayerFolders).To(Equal(expectedLayerDirs))
	})

	It("creates the volume store if it doesn't exist", func() {
		_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(d.VolumeStore()).To(BeADirectory())
	})

	It("uses hcs to create the volume", func() {
		_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
		Expect(err).ToNot(HaveOccurred())

		di, id, parentDir, allDirs := hcsClientFake.CreateLayerArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: d.VolumeStore(), Flavour: 1}))
		Expect(id).To(Equal(bundleID))

		expectedLayerDirs := []string{
			filepath.Join(d.LayerStore(), "newest-layer"),
			filepath.Join(d.LayerStore(), "middle-layer"),
			filepath.Join(d.LayerStore(), "oldest-layer"),
		}
		Expect(parentDir).To(Equal(expectedLayerDirs[0]))
		Expect(allDirs).To(Equal(expectedLayerDirs))
	})

	It("writes a base_image_size file", func() {
		_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
		Expect(err).ToNot(HaveOccurred())

		file := filepath.Join(d.VolumeStore(), bundleID, "base_image_size")
		contents, err := ioutil.ReadFile(file)
		Expect(err).ToNot(HaveOccurred())

		size, err := strconv.ParseInt(string(contents), 10, 64)
		Expect(err).ToNot(HaveOccurred())

		Expect(size).To(Equal(bundleSpec.BaseImageSize))
	})

	Context("no disk limit is given", func() {
		It("does not set a quota on the volume", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(limiterFake.SetQuotaCallCount()).To(Equal(0))
		})
	})

	Context("a disk limit is given", func() {
		BeforeEach(func() {
			bundleSpec.DiskLimit = 1234
		})

		It("sets the quota on the volume to the difference between the limit and the base size", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(limiterFake.SetQuotaCallCount()).To(Equal(1))
			vg, l := limiterFake.SetQuotaArgsForCall(0)
			Expect(vg).To(Equal(volumeGUID))
			Expect(l).To(Equal(uint64(1000)))
		})

		Context("the disk limit is less than the base image size", func() {
			BeforeEach(func() {
				bundleSpec.BaseImageSize = 2340
			})

			It("returns an error", func() {
				_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(&driver.DiskLimitTooSmallError{Limit: 1234, Base: 2340}))
			})
		})

		Context("the disk limit is equal to the base image size", func() {
			BeforeEach(func() {
				bundleSpec.BaseImageSize = 1234
			})

			It("returns an error", func() {
				_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(&driver.DiskLimitTooSmallError{Limit: 1234, Base: 1234}))
			})
		})

		Context("the limit is negative", func() {
			BeforeEach(func() {
				bundleSpec.DiskLimit = -1234
			})

			It("returns an error", func() {
				_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(&driver.InvalidDiskLimitError{Limit: -1234}))
			})
		})

		Context("ExcludeImageFromQuota is set", func() {
			BeforeEach(func() {
				bundleSpec.ExcludeImageFromQuota = true
			})

			It("sets the quota on the volume to the limit", func() {
				_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
				Expect(err).ToNot(HaveOccurred())
				Expect(limiterFake.SetQuotaCallCount()).To(Equal(1))
				vg, l := limiterFake.SetQuotaArgsForCall(0)
				Expect(vg).To(Equal(volumeGUID))
				Expect(l).To(Equal(uint64(1234)))
			})
		})
	})

	Context("a volume with the same id has already been created", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, true, nil)
		})

		It("returns a helpful error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(MatchError(&driver.LayerExistsError{Id: bundleID}))
		})
	})

	Context("checking if a volume of the same id exists errors", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, errors.New("LayerExists failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(MatchError("LayerExists failed"))
		})
	})

	Context("the driver store is unset", func() {
		BeforeEach(func() {
			d.Store = ""
		})

		It("return an error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("driver store must be set"))
		})
	})

	Context("creating the volume fails in hcs", func() {
		BeforeEach(func() {
			hcsClientFake.CreateLayerReturnsOnCall(0, errors.New("CreateLayer failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(MatchError("CreateLayer failed"))
		})
	})

	Context("getting the volume GUID fails in hcs", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", errors.New("GetLayerMountPath failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(MatchError("GetLayerMountPath failed"))
		})
	})

	Context("getting the volume GUID returns an empty value", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", nil)
		})

		It("returns a helpful error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs, bundleSpec)
			Expect(err).To(MatchError(&driver.MissingVolumePathError{Id: bundleID}))
		})
	})
})
