package driver_test

import (
	"errors"

	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	"code.cloudfoundry.org/lager/lagertest"
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

		d = driver.New("some-layer-store", "some-volume-store", hcsClientFake, tarStreamerFake, privilegeElevatorFake)
		logger = lagertest.NewTestLogger("driver-unpack-test")
		layerID = "some-layer-id"
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
