package driver_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

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
		logger                *lagertest.TestLogger
		layerIDs              = []string{"oldest-layer", "middle-layer", "newest-layer"}
	)

	BeforeEach(func() {
		var err error

		storeDir, err = ioutil.TempDir("", "bundle-store")
		Expect(err).NotTo(HaveOccurred())

		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}

		d = driver.New(filepath.Join(storeDir, driver.LayerDir),
			filepath.Join(storeDir, driver.VolumeDir),
			hcsClientFake, tarStreamerFake, privilegeElevatorFake)

		logger = lagertest.NewTestLogger("driver-unpack-test")
		hcsClientFake.GetLayerMountPathReturnsOnCall(0, volumeGUID, nil)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	It("returns a valid runtime spec", func() {
		spec, err := d.Bundle(logger, bundleID, layerIDs)
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
		_, err := d.Bundle(logger, bundleID, layerIDs)
		Expect(err).ToNot(HaveOccurred())
		Expect(d.VolumeStore()).To(BeADirectory())
	})

	It("uses hcs to create the volume", func() {
		_, err := d.Bundle(logger, bundleID, layerIDs)
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

	Context("a volume with the same id has already been created", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, true, nil)
		})

		It("returns a helpful error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs)
			Expect(err).To(MatchError(&driver.LayerExistsError{Id: bundleID}))
		})
	})

	Context("checking if a volume of the same id exists errors", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, errors.New("LayerExists failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs)
			Expect(err).To(MatchError("LayerExists failed"))
		})
	})

	Context("creating the volume fails in hcs", func() {
		BeforeEach(func() {
			hcsClientFake.CreateLayerReturnsOnCall(0, errors.New("CreateLayer failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs)
			Expect(err).To(MatchError("CreateLayer failed"))
		})
	})

	Context("getting the volume GUID fails in hcs", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", errors.New("GetLayerMountPath failed"))
		})

		It("returns the error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs)
			Expect(err).To(MatchError("GetLayerMountPath failed"))
		})
	})

	Context("getting the volume GUID returns an empty value", func() {
		BeforeEach(func() {
			hcsClientFake.GetLayerMountPathReturnsOnCall(0, "", nil)
		})

		It("returns a helpful error", func() {
			_, err := d.Bundle(logger, bundleID, layerIDs)
			Expect(err).To(MatchError(&driver.MissingVolumePathError{Id: bundleID}))
		})
	})
})
