package driver_test

import (
	"errors"

	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exists", func() {
	var (
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		logger                *lagertest.TestLogger
		layerID               string
	)

	BeforeEach(func() {
		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake)
		d.Store = "some-store-dir"

		logger = lagertest.NewTestLogger("driver-unpack-test")
		layerID = "some-layer-id"
	})

	It("passes the correct DriverInfo to LayerExists", func() {
		d.Exists(logger, layerID)
		Expect(hcsClientFake.LayerExistsCallCount()).To(Equal(1))
		di, id := hcsClientFake.LayerExistsArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: d.LayerStore(), Flavour: 1}))
		Expect(id).To(Equal(layerID))
	})

	Context("the layer has already been unpacked", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, true, nil)
		})

		It("returns true", func() {
			Expect(d.Exists(logger, layerID)).To(BeTrue())
		})
	})

	Context("the layer has not been unpacked", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, nil)
		})

		It("returns false", func() {
			Expect(d.Exists(logger, layerID)).To(BeFalse())
		})
	})

	Context("LayerExists returns an error", func() {
		BeforeEach(func() {
			hcsClientFake.LayerExistsReturnsOnCall(0, false, errors.New("LayerExists failed"))
		})

		It("returns false + logs an error", func() {
			Expect(d.Exists(logger, layerID)).To(BeFalse())
			Expect(logger.LogMessages()).To(ContainElement("driver-unpack-test.error-checking-layer"))
		})
	})
})
