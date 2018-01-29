package groot_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	errors "github.com/pkg/errors"
)

var _ = Describe("Create", func() {
	var (
		driver            *grootfakes.FakeDriver
		imagePuller       *grootfakes.FakeImagePuller
		driverRuntimeSpec = specs.Spec{Version: "some-version"}

		logger *lagertest.TestLogger
		g      *groot.Groot

		rootfsSource *url.URL
	)

	BeforeEach(func() {
		tempFile, err := ioutil.TempFile("", "groot-unit-tests")
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprint(tempFile, "afile")
		rootfsSource, err = url.Parse(tempFile.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(tempFile.Close()).To(Succeed())

		driver = new(grootfakes.FakeDriver)
		driver.BundleReturns(driverRuntimeSpec, nil)
		imagePuller = new(grootfakes.FakeImagePuller)

		imagePuller.PullReturns(imagepuller.Image{
			ChainIDs: []string{"checksum"},
		}, nil)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{
			Driver:      driver,
			Logger:      logger,
			ImagePuller: imagePuller,
		}
	})

	AfterEach(func() {
		Expect(os.Remove(rootfsSource.String())).To(Succeed())
	})

	Describe("Create succeeding", func() {
		var (
			returnedRuntimeSpec specs.Spec
			rootfsFileBuffer    *bytes.Buffer
		)

		BeforeEach(func() {
			rootfsFileBuffer = bytes.NewBuffer([]byte{})
			driver.UnpackStub = func(logger lager.Logger, id string, parentIDs []string, layerTar io.Reader) error {
				_, err := io.Copy(rootfsFileBuffer, layerTar)
				Expect(err).NotTo(HaveOccurred())
				return nil
			}
		})

		JustBeforeEach(func() {
			var err error
			returnedRuntimeSpec, err = g.Create("some-handle", rootfsSource)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls the image puller with the expected args", func() {
			Expect(imagePuller.PullCallCount()).To(Equal(1))
			_, spec := imagePuller.PullArgsForCall(0)
			Expect(spec.ImageSrc).To(Equal(rootfsSource))
		})

		It("returns the runtime spec from driver.Bundle", func() {
			Expect(returnedRuntimeSpec).To(Equal(driverRuntimeSpec))
		})

		It("calls driver.Bundle with the expected args", func() {
			Expect(driver.BundleCallCount()).To(Equal(1))
			_, id, layerIDs := driver.BundleArgsForCall(0)
			Expect(id).To(Equal("some-handle"))
			Expect(layerIDs).To(Equal([]string{"checksum"}))
		})

		Context("when the layer already exists", func() {
			BeforeEach(func() {
				driver.ExistsReturns(true)
			})

			It("doesn't call driver.Unpack", func() {
				Expect(driver.UnpackCallCount()).To(Equal(0))
			})
		})
	})

	Describe("Create failing", func() {
		var (
			createErr error
		)

		JustBeforeEach(func() {
			_, createErr = g.Create("some-handle", rootfsSource)
		})

		Context("when image puller returns an error", func() {
			BeforeEach(func() {
				imagePuller.PullReturns(imagepuller.Image{}, errors.New("pull-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("pull-failed")))
			})
		})

		Context("when driver.Bundle returns an error", func() {
			BeforeEach(func() {
				driver.BundleReturns(specs.Spec{}, errors.New("bundle-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("bundle-failed")))
			})
		})
	})
})
