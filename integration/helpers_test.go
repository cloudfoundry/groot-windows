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
	"unsafe"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot-windows/driver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"

	"github.com/Microsoft/hcsshim"
	"github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func grootPull(configFile, imageURI string) {
	pullCmd := exec.Command(grootBin, "--config", configFile, "pull", imageURI)
	_, _, err := execute(pullCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func grootCreate(configFile, imageURI, bundleID string) specs.Spec {
	createCmd := exec.Command(grootBin, "--config", configFile, "create", imageURI, bundleID)
	stdOut, _, err := execute(createCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var outputSpec specs.Spec
	ExpectWithOffset(1, json.Unmarshal(stdOut.Bytes(), &outputSpec)).To(Succeed())

	return outputSpec
}

func grootDelete(configFile, bundleID string) {
	deleteCmd := exec.Command(grootBin, "--config", configFile, "delete", bundleID)
	_, _, err := execute(deleteCmd)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func writeConfig(storeDir string) string {
	f, err := ioutil.TempFile("", "create.config")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	confFile := f.Name()
	_, err = f.Write([]byte(fmt.Sprintf("store: %s", storeDir)))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, f.Close()).To(Succeed())
	return confFile
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

func destroyLayerStore(configFile string) {
	layerStore := layerStoreFromConfig(configFile)

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

func layerStoreFromConfig(configFile string) string {
	var conf groot.Config

	contents, err := ioutil.ReadFile(configFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(yaml.Unmarshal(contents, &conf)).To(Succeed())
	return filepath.Join(conf.Store, driver.LayerDir)
}

func destroyVolumeStore(configFile string) {
	volumeStore := volumeStoreFromConfig(configFile)

	files, err := ioutil.ReadDir(volumeStore)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, f := range files {
		if f.IsDir() {
			grootDelete(configFile, f.Name())
		}
	}

	ExpectWithOffset(1, os.RemoveAll(volumeStore)).To(Succeed())
}

func volumeStoreFromConfig(configFile string) string {
	var conf groot.Config

	contents, err := ioutil.ReadFile(configFile)
	Expect(err).NotTo(HaveOccurred())
	Expect(yaml.Unmarshal(contents, &conf)).To(Succeed())
	return filepath.Join(conf.Store, driver.VolumeDir)
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
