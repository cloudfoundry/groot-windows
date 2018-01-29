package integration_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Create", func() {
	var (
		layerStore string
		imageURI   string
	)

	BeforeEach(func() {
		var err error
		layerStore, err = ioutil.TempDir("", "layer-store")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		destroyLayerStore(layerStore)
	})

	Context("provided an OCI image URI", func() {
		var ociImagePath string

		BeforeEach(func() {
			var err error
			ociImagePath, err = ioutil.TempDir("", "groot-windows-test-image")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(ociImagePath)).To(Succeed())
		})

		Context("when the image contains a layer with a regular file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
				Expect(extractTarGz(ociImageTgz, ociImagePath)).To(Succeed())
				imageURI = pathToOCIURI(ociImagePath)
			})

			It("unpacks the layer to disk", func() {
				createCmd := exec.Command(grootBin, "create", imageURI, randomContainerId())
				createCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_BASE_DIR=%s", layerStore))
				session, err := gexec.Start(createCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				chainIDs := getLayerChainIdsFromOCIImage(ociImagePath)
				knownFilePath := filepath.Join(layerStore, chainIDs[len(chainIDs)-1], "Files", "temp", "test", "hello")
				Expect(knownFilePath).To(BeAnExistingFile())
			})
		})

		Context("when the image contains a layer with a whiteout file", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-whiteout.tgz")
				Expect(extractTarGz(ociImageTgz, ociImagePath)).To(Succeed())
				imageURI = pathToOCIURI(ociImagePath)
			})

			It("unpacks the layer to disk and the whited-out file does not exist in the final layer", func() {
				createCmd := exec.Command(grootBin, "create", imageURI, randomContainerId())
				createCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_BASE_DIR=%s", layerStore))
				session, err := gexec.Start(createCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				chainIDs := getLayerChainIdsFromOCIImage(ociImagePath)
				knownFilePathFileExists := filepath.Join(layerStore, chainIDs[len(chainIDs)-1], "Files", "temp", "test", "hello2")
				Expect(knownFilePathFileExists).To(BeAnExistingFile())
				knownFilePathFileRemoved := filepath.Join(layerStore, chainIDs[len(chainIDs)-1], "Files", "temp", "test", "hello")
				Expect(knownFilePathFileRemoved).ToNot(BeAnExistingFile())
			})
		})

		Context("when the image contains a layer with symlinks and hardlinks", func() {
			BeforeEach(func() {
				ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-link.tgz")
				Expect(extractTarGz(ociImageTgz, ociImagePath)).To(Succeed())
				imageURI = pathToOCIURI(ociImagePath)
			})

			It("unpacks the layer to disk with the correct links intact", func() {
				createCmd := exec.Command(grootBin, "create", imageURI, randomContainerId())
				createCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_BASE_DIR=%s", layerStore))
				session, err := gexec.Start(createCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))

				chainIDs := getLayerChainIdsFromOCIImage(ociImagePath)

				symlinkFilePath := filepath.Join(layerStore, chainIDs[len(chainIDs)-4], "Files", "temp", "symlinkfile")
				dest, err := os.Readlink(symlinkFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(dest).To(Equal("C:\\temp\\test\\hello"))

				hardlinkFilePath := filepath.Join(layerStore, chainIDs[len(chainIDs)-3], "Files", "temp", "hardlinkfile")
				data, err := ioutil.ReadFile(hardlinkFilePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.TrimSpace(string(data))).To(Equal("hello"))

				symlinkDirPath := filepath.Join(layerStore, chainIDs[len(chainIDs)-2], "Files", "temp", "symlinkdir")
				Expect(getReparseTag(symlinkDirPath)).To(Equal(uint32(syscall.IO_REPARSE_TAG_SYMLINK)), "not a symlink")
				Expect(getSymlinkDest(symlinkDirPath)).To(Equal("C:\\temp\\test"))
				Expect(getFileAttributes(symlinkDirPath)&syscall.FILE_ATTRIBUTE_DIRECTORY).To(Equal(uint32(syscall.FILE_ATTRIBUTE_DIRECTORY)), "not a directory")

				junctionDirPath := filepath.Join(layerStore, chainIDs[len(chainIDs)-1], "Files", "temp", "junctiondir")
				Expect(getReparseTag(junctionDirPath)).To(Equal(uint32(IO_REPARSE_TAG_MOUNT_POINT)), "not a junction point")
				Expect(getSymlinkDest(junctionDirPath)).To(Equal("C:\\temp\\test"))
				Expect(getFileAttributes(junctionDirPath)&syscall.FILE_ATTRIBUTE_DIRECTORY).To(Equal(uint32(syscall.FILE_ATTRIBUTE_DIRECTORY)), "not a directory")
			})
		})
	})

	Context("provided a Docker image URI", func() {
		var (
			ociImagePath  string
			knownFilePath string
		)

		BeforeEach(func() {
			imageURI = "docker:///pivotalgreenhouse/groot-windows-test:regularfile"

			var err error
			ociImagePath, err = ioutil.TempDir("", "groot-windows-test-image")
			Expect(err).ToNot(HaveOccurred())

			// we need to know the layer ids of our image and we can find that out from this fixture
			ociImageTgz := filepath.Join(imageTgzDir, "groot-windows-test-regularfile.tgz")
			Expect(extractTarGz(ociImageTgz, ociImagePath)).To(Succeed())
			chainIDs := getLayerChainIdsFromOCIImage(ociImagePath)
			knownFilePath = filepath.Join(layerStore, chainIDs[len(chainIDs)-1], "Files", "temp", "test", "hello")
			Expect(os.RemoveAll(ociImagePath)).To(Succeed())
		})

		It("unpacks the layer to disk", func() {
			createCmd := exec.Command(grootBin, "create", imageURI, randomContainerId())
			createCmd.Env = append(os.Environ(), fmt.Sprintf("GROOT_BASE_DIR=%s", layerStore))
			session, err := gexec.Start(createCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(knownFilePath).To(BeAnExistingFile())
		})
	})
})

func randomContainerId() string {
	max := big.NewInt(math.MaxInt64)
	r, err := rand.Int(rand.Reader, max)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return fmt.Sprintf("%d", r.Int64())
}

func pathToOCIURI(path string) string {
	return fmt.Sprintf("oci:///%s", filepath.ToSlash(path))
}

func getLayerChainIdsFromOCIImage(imagePath string) []string {
	indexFile, err := os.Open(filepath.Join(imagePath, "index.json"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer indexFile.Close()

	var index v1.Index
	indexDec := json.NewDecoder(indexFile)
	ExpectWithOffset(1, indexDec.Decode(&index)).To(Succeed())
	ExpectWithOffset(1, index.Manifests).ToNot(BeEmpty())
	manifestDigest := strings.TrimPrefix(index.Manifests[0].Digest.String(), "sha256:")

	manifestFile, err := os.Open(filepath.Join(imagePath, "blobs", "sha256", manifestDigest))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer manifestFile.Close()

	var manifest v1.Manifest
	manifestDec := json.NewDecoder(manifestFile)
	ExpectWithOffset(1, manifestDec.Decode(&manifest)).To(Succeed())
	configDigest := strings.TrimPrefix(manifest.Config.Digest.String(), "sha256:")

	configFile, err := os.Open(filepath.Join(imagePath, "blobs", "sha256", configDigest))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	defer configFile.Close()

	var config v1.Image
	configDec := json.NewDecoder(configFile)
	ExpectWithOffset(1, configDec.Decode(&config)).To(Succeed())

	diffIDs := []string{}
	for _, id := range config.RootFS.DiffIDs {
		diffIDs = append(diffIDs, strings.TrimPrefix(id.String(), "sha256:"))
	}

	chainIDs := []string{}
	parentChainID := ""
	for _, diffID := range diffIDs {
		chainID := diffID

		if parentChainID != "" {
			chainIDSha := sha256.Sum256([]byte(fmt.Sprintf("%s %s", parentChainID, diffID)))
			chainID = hex.EncodeToString(chainIDSha[:32])
		}

		parentChainID = chainID

		chainIDs = append(chainIDs, chainID)
	}

	return chainIDs
}

func destroyLayerStore(layerStore string) {
	files, err := ioutil.ReadDir(layerStore)
	Expect(err).ToNot(HaveOccurred())

	di := hcsshim.DriverInfo{HomeDir: layerStore, Flavour: 1}
	for _, f := range files {
		if f.IsDir() {
			Expect(hcsshim.DestroyLayer(di, filepath.Base(f.Name()))).To(Succeed())
		}
	}

	Expect(os.RemoveAll(layerStore)).To(Succeed())
}

func extractTarGz(tarfile, destDir string) error {
	file, err := os.Open(tarfile)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(gz, destDir)
}

func extractTar(src io.Reader, destDir string) error {
	tr := tar.NewReader(src)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		path := filepath.Join(destDir, hdr.Name)
		fi := hdr.FileInfo()

		if fi.IsDir() {
			err = os.MkdirAll(path, hdr.FileInfo().Mode())
		} else if fi.Mode()&os.ModeSymlink != 0 {
			target := hdr.Linkname
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err = os.Symlink(target, path); err != nil {
				return err
			}
		} else {
			err = writeToFile(tr, path, hdr.FileInfo().Mode())
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}

const IO_REPARSE_TAG_MOUNT_POINT = 0xA0000003

type reparseDataBuffer struct {
	ReparseTag        uint32
	ReparseDataLength uint16
	Reserved          uint16

	reparseBuffer byte
}

type symbolicLinkReparseBuffer struct {
	SubstituteNameOffset uint16
	SubstituteNameLength uint16
	PrintNameOffset      uint16
	PrintNameLength      uint16
	Flags                uint32
	PathBuffer           [1]uint16
}

type mountPointReparseBuffer struct {
	SubstituteNameOffset uint16
	SubstituteNameLength uint16
	PrintNameOffset      uint16
	PrintNameLength      uint16
	PathBuffer           [1]uint16
}

func getSymlinkDest(filename string) string {
	fd := openSymlinkDir(filename)
	defer syscall.CloseHandle(fd)

	rdbbuf := make([]byte, syscall.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var bytesReturned uint32
	Expect(syscall.DeviceIoControl(fd, syscall.FSCTL_GET_REPARSE_POINT, nil, 0, &rdbbuf[0], uint32(len(rdbbuf)), &bytesReturned, nil)).To(Succeed())

	rdb := (*reparseDataBuffer)(unsafe.Pointer(&rdbbuf[0]))

	var s string
	switch rdb.ReparseTag {
	case syscall.IO_REPARSE_TAG_SYMLINK:
		data := (*symbolicLinkReparseBuffer)(unsafe.Pointer(&rdb.reparseBuffer))
		p := (*[0xffff]uint16)(unsafe.Pointer(&data.PathBuffer[0]))
		s = syscall.UTF16ToString(p[data.SubstituteNameOffset/2 : (data.SubstituteNameOffset+data.SubstituteNameLength)/2])

	case IO_REPARSE_TAG_MOUNT_POINT:
		data := (*mountPointReparseBuffer)(unsafe.Pointer(&rdb.reparseBuffer))
		p := (*[0xffff]uint16)(unsafe.Pointer(&data.PathBuffer[0]))
		s = syscall.UTF16ToString(p[data.SubstituteNameOffset/2 : (data.SubstituteNameOffset+data.SubstituteNameLength)/2])
	default:
		panic(fmt.Sprintf("unknown reparse tag %d", rdb.ReparseTag))
	}

	return strings.Replace(s, `\??\`, "", -1)
}

func getReparseTag(filename string) uint32 {
	fd := openSymlinkDir(filename)
	defer syscall.CloseHandle(fd)

	rdbbuf := make([]byte, syscall.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var bytesReturned uint32
	Expect(syscall.DeviceIoControl(fd, syscall.FSCTL_GET_REPARSE_POINT, nil, 0, &rdbbuf[0], uint32(len(rdbbuf)), &bytesReturned, nil)).To(Succeed())

	rdb := (*reparseDataBuffer)(unsafe.Pointer(&rdbbuf[0]))
	return rdb.ReparseTag
}

func getFileAttributes(filename string) uint32 {
	fd := openSymlinkDir(filename)
	defer syscall.CloseHandle(fd)

	var d syscall.ByHandleFileInformation
	Expect(syscall.GetFileInformationByHandle(fd, &d)).To(Succeed())
	return d.FileAttributes
}

func openSymlinkDir(filename string) syscall.Handle {
	fd, err := syscall.CreateFile(syscall.StringToUTF16Ptr(filename), 0, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	Expect(err).NotTo(HaveOccurred())
	return fd
}
