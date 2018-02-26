package driver_test

import (
	"errors"
	"path/filepath"

	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		limiterFake           *fakes.Limiter
		logger                *lagertest.TestLogger
		bundleID              string
	)

	BeforeEach(func() {
		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}
		limiterFake = &fakes.Limiter{}

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake, limiterFake)
		d.Store = "some-store-dir"

		logger = lagertest.NewTestLogger("driver-delete-test")
		bundleID = "some-bundle-id"

		hcsClientFake.LayerExistsReturnsOnCall(0, true, nil)
	})

	It("checks the volume's existence and deletes it", func() {
		Expect(d.Delete(logger, bundleID)).To(Succeed())

		Expect(hcsClientFake.LayerExistsCallCount()).To(Equal(1))
		di, id := hcsClientFake.LayerExistsArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: filepath.Join("C:\\some-store-dir", "volumes"), Flavour: 1}))
		Expect(id).To(Equal("some-bundle-id"))

		Expect(hcsClientFake.DestroyLayerCallCount()).To(Equal(1))
		di, id = hcsClientFake.DestroyLayerArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: filepath.Join("C:\\some-store-dir", "volumes"), Flavour: 1}))
		Expect(id).To(Equal("some-bundle-id"))
	})

	Context("delete fails", func() {
		BeforeEach(func() {
			hcsClientFake.DestroyLayerReturnsOnCall(0, errors.New("Destroy layer failed"))
		})

		It("returns the error", func() {
			Expect(d.Delete(logger, bundleID)).To(MatchError("Destroy layer failed"))
		})
	})

	Context("the volume doesn't exist", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, nil)
		})

		It("logs the error and returns success", func() {
			Expect(d.Delete(logger, bundleID)).To(Succeed())
			Expect(logger.LogMessages()).To(ContainElement("driver-delete-test.volume-not-found"))
		})
	})

	Context("checking the volume's existence fails", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, errors.New("Layer exists failed"))
		})

		It("returns the error", func() {
			Expect(d.Delete(logger, bundleID)).To(MatchError("Layer exists failed"))
		})
	})

	Context("the driver store is unset", func() {
		BeforeEach(func() {
			d.Store = ""
		})

		It("return an error", func() {
			err := d.Delete(logger, bundleID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("driver store must be set"))
		})
	})
})
