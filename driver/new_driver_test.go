package driver_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("New Driver", func() {
	var (
		creator  *driver.Creator
		config   groot.Config
		storeDir string
	)

	BeforeEach(func() {
		var err error
		storeDir, err = ioutil.TempDir("", "new-driver.store")
		Expect(err).NotTo(HaveOccurred())

		creator = &driver.Creator{}

		config = groot.Config{Store: storeDir}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	It("creates a driver with the correct layer store and volume store", func() {
		d, err := creator.NewDriver(config)
		Expect(err).NotTo(HaveOccurred())

		winDriver, ok := d.(*driver.Driver)
		Expect(ok).To(BeTrue())
		Expect(winDriver.LayerStore()).To(Equal(filepath.Join(storeDir, driver.LayerDir)))
		Expect(winDriver.VolumeStore()).To(Equal(filepath.Join(storeDir, driver.VolumeDir)))
	})

	It("creates the layer store and the volume store if they don't exist", func() {
		d, err := creator.NewDriver(config)
		Expect(err).NotTo(HaveOccurred())

		winDriver, ok := d.(*driver.Driver)
		Expect(ok).To(BeTrue())
		Expect(winDriver.LayerStore()).To(BeADirectory())
		Expect(winDriver.VolumeStore()).To(BeADirectory())
	})

	Context("store dir is not specified", func() {
		BeforeEach(func() {
			config = groot.Config{}
		})

		It("returns a helpful error", func() {
			_, err := creator.NewDriver(config)
			Expect(err).To(MatchError("must set store"))
		})
	})
})
