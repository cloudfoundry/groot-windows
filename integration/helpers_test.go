package integration_test

import (
	"archive/tar"
	"bytes"
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
	"unicode/utf16"
	"unsafe"

	"code.cloudfoundry.org/groot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Microsoft/hcsshim"
	"github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func grootPull(driverStore, imageURI string) {
	pullCmd := exec.Command(grootBin, "--driver-store", driverStore, "pull", imageURI)
	_, _, err := execute(pullCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func grootCreate(driverStore, imageURI, bundleID string, createArgs ...string) specs.Spec {
	createCmd := exec.Command(grootBin, "--driver-store", driverStore, "create")
	createCmd.Args = append(createCmd.Args, createArgs...)
	createCmd.Args = append(createCmd.Args, imageURI, bundleID)
	stdOut, _, err := execute(createCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var outputSpec specs.Spec
	ExpectWithOffset(1, json.Unmarshal(stdOut.Bytes(), &outputSpec)).To(Succeed())

	return outputSpec
}

func grootDelete(driverStore, bundleID string) {
	deleteCmd := exec.Command(grootBin, "--driver-store", driverStore, "delete", bundleID)
	_, _, err := execute(deleteCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func grootStats(driverStore, bundleID string) groot.VolumeStats {
	statsCmd := exec.Command(grootBin, "--driver-store", driverStore, "stats", bundleID)
	stdout, _, err := execute(statsCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var stats groot.VolumeStats
	ExpectWithOffset(1, json.Unmarshal(stdout.Bytes(), &stats)).To(Succeed())
	return stats
}

func execute(c *exec.Cmd) (*bytes.Buffer, *bytes.Buffer, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	c.Stdout = io.MultiWriter(stdOut, GinkgoWriter)
	c.Stderr = io.MultiWriter(stdErr, GinkgoWriter)
	err := c.Run()

	return stdOut, stdErr, err
}

func mountVolume(volumeGuid, mountPath string) {
	ExpectWithOffset(1, exec.Command("mountvol", mountPath, volumeGuid).Run()).To(Succeed())
}

func unmountVolume(mountPath string) {
	if _, _, err := execute(exec.Command("mountvol", mountPath, "/L")); err != nil {
		return
	}

	_, _, err := execute(exec.Command("mountvol", mountPath, "/D"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, os.RemoveAll(mountPath)).To(Succeed())
}

func randomBundleID() string {
	max := big.NewInt(math.MaxInt64)
	r, err := rand.Int(rand.Reader, max)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return fmt.Sprintf("%d", r.Int64())
}
func pathToUnixURI(path string) string {
	path = strings.Replace(path, "C:", "", 1)
	path = strings.Replace(path, `\`, `/`, -1)
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("oci:///%s", path)
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

func destroyLayerStore(driverStore string) {
	layerStore := filepath.Join(driverStore, "layers")
	_, err := os.Stat(layerStore)
	if err != nil && os.IsNotExist(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	files, err := ioutil.ReadDir(layerStore)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	di := hcsshim.DriverInfo{HomeDir: layerStore, Flavour: 1}
	for _, f := range files {
		if f.IsDir() {
			ExpectWithOffset(1, hcsshim.DestroyLayer(di, filepath.Base(f.Name()))).To(Succeed())
		}
	}

	ExpectWithOffset(1, os.RemoveAll(layerStore)).To(Succeed())
}

func destroyVolumeStore(driverStore string) {
	volumeStore := filepath.Join(driverStore, "volumes")
	_, err := os.Stat(volumeStore)
	if err != nil && os.IsNotExist(err) {
		return
	}
	Expect(err).NotTo(HaveOccurred())

	files, err := ioutil.ReadDir(volumeStore)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, f := range files {
		if f.IsDir() {
			grootDelete(driverStore, f.Name())
		}
	}

	ExpectWithOffset(1, os.RemoveAll(volumeStore)).To(Succeed())
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
	ExpectWithOffset(1, syscall.DeviceIoControl(fd, syscall.FSCTL_GET_REPARSE_POINT, nil, 0, &rdbbuf[0], uint32(len(rdbbuf)), &bytesReturned, nil)).To(Succeed())

	rdb := (*reparseDataBuffer)(unsafe.Pointer(&rdbbuf[0]))

	var s string
	switch rdb.ReparseTag {
	case syscall.IO_REPARSE_TAG_SYMLINK:
		data := (*symbolicLinkReparseBuffer)(unsafe.Pointer(&rdb.reparseBuffer))
		s = utf16PtrToString(&data.PathBuffer[0])
	case IO_REPARSE_TAG_MOUNT_POINT:
		data := (*mountPointReparseBuffer)(unsafe.Pointer(&rdb.reparseBuffer))
		s = utf16PtrToString(&data.PathBuffer[0])
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
	ExpectWithOffset(1, syscall.DeviceIoControl(fd, syscall.FSCTL_GET_REPARSE_POINT, nil, 0, &rdbbuf[0], uint32(len(rdbbuf)), &bytesReturned, nil)).To(Succeed())

	rdb := (*reparseDataBuffer)(unsafe.Pointer(&rdbbuf[0]))
	return rdb.ReparseTag
}

func getFileAttributes(filename string) uint32 {
	fd := openSymlinkDir(filename)
	defer syscall.CloseHandle(fd)

	var d syscall.ByHandleFileInformation
	ExpectWithOffset(1, syscall.GetFileInformationByHandle(fd, &d)).To(Succeed())
	return d.FileAttributes
}

func openSymlinkDir(filename string) syscall.Handle {
	fd, err := syscall.CreateFile(syscall.StringToUTF16Ptr(filename), 0, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return fd
}

func getLastWriteTime(file string) int64 {
	fi, err := os.Stat(file)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return fi.Sys().(*syscall.Win32FileAttributeData).LastWriteTime.Nanoseconds()
}

// Taken from: .../go/1.16.3/libexec/src/internal/syscall/windows/syscall_windows.go
// UTF16PtrToString is like UTF16ToString, but takes *uint16
// as a parameter instead of []uint16.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	// Find NUL terminator.
	end := unsafe.Pointer(p)
	n := 0
	for *(*uint16)(end) != 0 {
		end = unsafe.Pointer(uintptr(end) + unsafe.Sizeof(*p))
		n++
	}
	// Turn *uint16 into []uint16.
	var s []uint16
	hdr := (*unsafeheaderSlice)(unsafe.Pointer(&s))
	hdr.Data = unsafe.Pointer(p)
	hdr.Cap = n
	hdr.Len = n
	// Decode []uint16 into string.
	return string(utf16.Decode(s))
}

// Taken from: .../go/1.16.3/libexec/src/internal/unsafeheader/unsafeheader.go
// Slice is the runtime representation of a slice.
// It cannot be used safely or portably and its representation may
// change in a later release.
//
// Unlike reflect.SliceHeader, its Data field is sufficient to guarantee the
// data it references will not be garbage collected.
type unsafeheaderSlice struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}
