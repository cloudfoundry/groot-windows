package driver_test

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot-windows/driver"
	"code.cloudfoundry.org/groot-windows/driver/fakes"
	hcsfakes "code.cloudfoundry.org/groot-windows/hcs/fakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	winio "github.com/Microsoft/go-winio"
	"github.com/Microsoft/go-winio/archive/tar"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unpack", func() {
	var (
		storeDir              string
		d                     *driver.Driver
		hcsClientFake         *fakes.HCSClient
		tarStreamerFake       *fakes.TarStreamer
		privilegeElevatorFake *fakes.PrivilegeElevator
		limiterFake           *fakes.Limiter
		logger                lager.Logger
		layerID               string
		buffer                *bytes.Buffer
		layerWriterFake       *hcsfakes.LayerWriter
	)

	BeforeEach(func() {
		var err error
		storeDir, err = ioutil.TempDir("", "driver")
		Expect(err).To(Succeed())

		hcsClientFake = &fakes.HCSClient{}
		tarStreamerFake = &fakes.TarStreamer{}
		privilegeElevatorFake = &fakes.PrivilegeElevator{}
		limiterFake = &fakes.Limiter{}

		d = driver.New(hcsClientFake, tarStreamerFake, privilegeElevatorFake, limiterFake)
		d.Store = storeDir

		logger = lagertest.NewTestLogger("driver-unpack-test")
		layerID = "aaa"
		buffer = bytes.NewBuffer([]byte("tar ball contents"))

		tarStreamerFake.NextReturns(nil, io.EOF)
		tarStreamerFake.WriteBackupStreamFromTarFileReturns(nil, io.EOF)

		layerWriterFake = &hcsfakes.LayerWriter{}
		hcsClientFake.NewLayerWriterReturns(layerWriterFake, nil)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storeDir)).To(Succeed())
	})

	It("create an associated layerId path", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

		expectedDir := filepath.Join(d.LayerStore(), layerID)
		Expect(expectedDir).To(BeADirectory())
	})

	It("elevates itself with the backup and restore privileges", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

		Expect(privilegeElevatorFake.EnableProcessPrivilegesCallCount()).To(Equal(1))
		Expect(privilegeElevatorFake.EnableProcessPrivilegesArgsForCall(0)).To(Equal([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}))
	})

	Context("when the backup/restore privileges cannot be acquired", func() {
		var expectedErr error

		BeforeEach(func() {
			expectedErr = errors.New("Failed to elevate privileges")
			privilegeElevatorFake.EnableProcessPrivilegesReturns(expectedErr)
		})

		It("errors", func() {
			Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
		})
	})

	It("releases the backup and restore privileges on exit", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

		Expect(privilegeElevatorFake.DisableProcessPrivilegesCallCount()).To(Equal(1))
		Expect(privilegeElevatorFake.DisableProcessPrivilegesArgsForCall(0)).To(Equal([]string{winio.SeBackupPrivilege, winio.SeRestorePrivilege}))
	})

	It("creates a layer writer with the correct layer id", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

		Expect(hcsClientFake.NewLayerWriterCallCount()).To(Equal(1))
		di, actualLayerID, parentIDs := hcsClientFake.NewLayerWriterArgsForCall(0)
		Expect(di).To(Equal(hcsshim.DriverInfo{HomeDir: d.LayerStore(), Flavour: 1}))
		Expect(actualLayerID).To(Equal(layerID))
		Expect(parentIDs).To(BeEmpty())
	})

	It("closes the layer writer on exit", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())
		Expect(layerWriterFake.CloseCallCount()).To(Equal(1))
	})

	It("sets up a tar reader with the layer tarball contents, clearing it at the end", func() {
		Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

		Expect(tarStreamerFake.SetReaderCallCount()).To(Equal(2))
		Expect(tarStreamerFake.SetReaderArgsForCall(0)).To(Equal(buffer))

		r := tarStreamerFake.SetReaderArgsForCall(1)
		b, ok := r.(*bytes.Reader)
		Expect(ok).To(BeTrue())
		Expect(b.Size()).To(Equal(int64(0)))
	})

	Context("when the layer contains files", func() {
		var (
			whiteoutFileHeader, linkFileHeader, regularFileHeader *tar.Header
		)

		BeforeEach(func() {
			whiteoutFileHeader = &tar.Header{Name: "something/somethingelse/.wh.filename"}
			linkFileHeader = &tar.Header{
				Name:     "something/somethingelse/linkfile",
				Typeflag: tar.TypeLink,
				Linkname: "link/name/file",
			}
			regularFileHeader = &tar.Header{Name: "regular/file/name"}
		})

		Context("the driver store is unset", func() {
			BeforeEach(func() {
				d.Store = ""
			})

			It("return an error", func() {
				err := d.Unpack(logger, layerID, []string{}, buffer)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("driver store must be set"))
			})
		})

		Context("when there are multiple files", func() {
			BeforeEach(func() {
				tarStreamerFake.NextReturnsOnCall(0, whiteoutFileHeader, nil)
				tarStreamerFake.NextReturnsOnCall(1, linkFileHeader, nil)
				tarStreamerFake.NextReturnsOnCall(2, regularFileHeader, nil)
				tarStreamerFake.NextReturnsOnCall(3, linkFileHeader, nil)

				tarStreamerFake.WriteBackupStreamFromTarFileReturnsOnCall(0, whiteoutFileHeader, nil)

				tarStreamerFake.FileInfoFromHeaderReturns("regular/file/name", 100, &winio.FileBasicInfo{}, nil)
			})

			It("reads files from the layer tarball until EOF", func() {
				Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

				Expect(tarStreamerFake.NextCallCount()).To(Equal(5))
			})
		})

		Context("the file is a whiteout file", func() {
			BeforeEach(func() {
				tarStreamerFake.NextReturnsOnCall(0, &tar.Header{
					Name: "something/somethingelse/.wh.filename",
				}, nil)
			})

			It("removes the file and finds the next file", func() {
				Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

				Expect(tarStreamerFake.NextCallCount()).To(Equal(2))

				Expect(layerWriterFake.RemoveCallCount()).To(Equal(1))
				Expect(layerWriterFake.RemoveArgsForCall(0)).To(Equal("something\\somethingelse\\filename"))
			})

			Context("when removing the file fails", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("Failed to remove file!")
					layerWriterFake.RemoveReturns(expectedErr)
				})

				It("errors", func() {
					Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
				})
			})
		})

		Context("the file is a link", func() {
			BeforeEach(func() {
				tarStreamerFake.NextReturnsOnCall(0, &tar.Header{
					Name:     "something/somethingelse/linkfile",
					Typeflag: tar.TypeLink,
					Linkname: "link/name/file",
				}, nil)
			})

			It("adds the file as a link", func() {
				Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

				Expect(tarStreamerFake.NextCallCount()).To(Equal(2))

				Expect(layerWriterFake.AddLinkCallCount()).To(Equal(1))
				nameArg, linknameArg := layerWriterFake.AddLinkArgsForCall(0)
				Expect(nameArg).To(Equal("something\\somethingelse\\linkfile"))
				Expect(linknameArg).To(Equal("link\\name\\file"))
			})

			Context("when adding the link fails", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("Failed to add link")
					layerWriterFake.AddLinkReturns(expectedErr)
				})

				It("errors", func() {
					Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
				})
			})
		})

		Context("the file is regular file", func() {
			var (
				tarHeader *tar.Header
				fileInfo  *winio.FileBasicInfo
			)

			BeforeEach(func() {
				tarHeader = &tar.Header{
					Name: "regular/file/name",
				}
				tarStreamerFake.NextReturnsOnCall(0, tarHeader, nil)
				fileInfo = &winio.FileBasicInfo{}
				tarStreamerFake.FileInfoFromHeaderReturns("regular/file/name", 100, fileInfo, nil)
			})

			It("adds the file to the layer", func() {
				Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(Succeed())

				Expect(tarStreamerFake.NextCallCount()).To(Equal(1))

				Expect(tarStreamerFake.FileInfoFromHeaderCallCount()).To(Equal(1))
				Expect(tarStreamerFake.FileInfoFromHeaderArgsForCall(0)).To(Equal(tarHeader))

				Expect(layerWriterFake.AddCallCount()).To(Equal(1))
				actualName, actualFileInfo := layerWriterFake.AddArgsForCall(0)
				Expect(actualName).To(Equal("regular\\file\\name"))
				Expect(actualFileInfo).To(Equal(fileInfo))

				Expect(tarStreamerFake.WriteBackupStreamFromTarFileCallCount()).To(Equal(1))
				actualWriter, actualTarHeader := tarStreamerFake.WriteBackupStreamFromTarFileArgsForCall(0)
				Expect(actualWriter).To(Equal(layerWriterFake))
				Expect(actualTarHeader).To(Equal(tarHeader))
			})

			Context("when getting the file info fails", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("Failed to get file info")
					tarStreamerFake.FileInfoFromHeaderReturns("", 0, nil, expectedErr)
				})

				It("errors", func() {
					Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
				})
			})

			Context("when adding the file to the layer fails", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("Failed to add file")
					layerWriterFake.AddReturns(expectedErr)
				})

				It("errors", func() {
					Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
				})
			})
		})

		Context("when getting the next file fails", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("Failed to get next file")
				tarStreamerFake.NextReturns(nil, expectedErr)
			})

			It("errors", func() {
				Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
			})
		})
	})

	Context("when the layer being unpacked has parents", func() {
		It("creates a layer writer with its parent layer paths from newest to oldest", func() {
			parentIDs := []string{"oldest-parent-id", "newest-parent-id"}
			Expect(d.Unpack(logger, layerID, parentIDs, buffer)).To(Succeed())

			_, _, hcsParentIds := hcsClientFake.NewLayerWriterArgsForCall(0)
			Expect(hcsParentIds).To(Equal([]string{filepath.Join(d.LayerStore(), "newest-parent-id"), filepath.Join(d.LayerStore(), "oldest-parent-id")}))
		})
	})

	Context("when creating the layer writer fails", func() {
		var expectedErr error

		BeforeEach(func() {
			expectedErr = errors.New("Failed to create layer writer!")
			hcsClientFake.NewLayerWriterReturns(nil, expectedErr)
		})

		It("errors", func() {
			Expect(d.Unpack(logger, layerID, []string{}, buffer)).To(MatchError(expectedErr))
		})
	})
})
