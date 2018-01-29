package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Extract", func() {
	var (
		outputDir   string
		extractSess *gexec.Session
		extractArgs []string

		imageLayers = []string{
			"67903cf26ef4095868687002e3dc6f78ad275677704bf0d11524f16209cec48e",
			"87f40e4c0087014df9322d0d046879e0acc6583d30d8aff4def445c11ec74cd1",
			"3c19ca70b62d5c7afc6bfb79e2203b467c1f831a51e1be87b4916d8cb593034a",
			"6b48ca8a17606fc0074fce44e4bc9a69e640a9a68932e42984f2000190a650be",
		}
	)

	BeforeEach(func() {
		var err error
		outputDir, err = ioutil.TempDir("", "extract.integration.test")
		Expect(err).NotTo(HaveOccurred())
		extractArgs = []string{rootfsTgz, outputDir}
	})

	JustBeforeEach(func() {
		var err error
		cmd := exec.Command(extractBin, extractArgs...)
		extractSess, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		for _, layer := range imageLayers {
			Expect(hcsshim.DestroyLayer(hcsshim.DriverInfo{HomeDir: outputDir, Flavour: 1}, layer)).To(Succeed())
		}
		Expect(os.RemoveAll(outputDir)).To(Succeed())
	})

	It("extracts a rootfs tarball to the output directory", func() {
		Eventually(extractSess).Should(gexec.Exit(0))
		for _, layer := range imageLayers {
			Expect(filepath.Join(outputDir, layer)).To(BeADirectory())
			data, err := ioutil.ReadFile(filepath.Join(outputDir, layer, ".complete"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(layer))
		}
		Expect(filepath.Join(outputDir, imageLayers[2], "Files", "ProgramData", "out.txt")).To(BeAnExistingFile())
		Expect(filepath.Join(outputDir, imageLayers[3], "Files", "ProgramData", "out1.txt")).To(BeAnExistingFile())
	})

	It("only outputs the top layer path to stdout", func() {
		Eventually(extractSess).Should(gexec.Exit(0))
		Expect(string(extractSess.Out.Contents())).To(Equal(filepath.Join(outputDir, imageLayers[3])))
	})

	Context("when an image has a layer that has been partially extracted", func() {
		var originalCreateTime int64

		BeforeEach(func() {
			cmd := exec.Command(extractBin, rootfsTgz, outputDir)
			firstSess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(firstSess).Should(gexec.Exit(0))
			Expect(os.Remove(filepath.Join(outputDir, imageLayers[3], "Files", "ProgramData", "out1.txt"))).To(Succeed())
			Expect(os.Remove(filepath.Join(outputDir, imageLayers[3], ".complete"))).To(Succeed())
			originalCreateTime = getCreatedTime(filepath.Join(outputDir, imageLayers[3]))
		})

		It("re-extracts the layer", func() {
			Eventually(extractSess).Should(gexec.Exit(0))
			Expect(filepath.Join(outputDir, imageLayers[3], "Files", "ProgramData", "out1.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(outputDir, imageLayers[3], ".complete")).To(BeAnExistingFile())
			Expect(getCreatedTime(filepath.Join(outputDir, imageLayers[3]))).To(BeNumerically(">", originalCreateTime))
		})
	})

	Context("when an image has a layer that has been fully extracted", func() {
		var originalCreateTimes []int64

		BeforeEach(func() {
			cmd := exec.Command(extractBin, rootfsTgz, outputDir)
			firstSess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(firstSess).Should(gexec.Exit(0))
			for _, layer := range imageLayers {
				originalCreateTimes = append(originalCreateTimes, getCreatedTime(filepath.Join(outputDir, layer)))
			}
		})

		It("does not modify the layer", func() {
			Eventually(extractSess).Should(gexec.Exit(0))
			Expect(filepath.Join(outputDir, imageLayers[3], "Files", "ProgramData", "out1.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(outputDir, imageLayers[3], ".complete")).To(BeAnExistingFile())

			for i, layer := range imageLayers {
				Expect(getCreatedTime(filepath.Join(outputDir, layer))).To(Equal(originalCreateTimes[i]))
			}
		})
	})

	Context("when not provided rootfs tgz and output dir arguments", func() {
		BeforeEach(func() {
			extractArgs = []string{"some-arg"}
		})

		It("fails with a helpful error message", func() {
			Eventually(extractSess).Should(gexec.Exit(1))
			Expect(extractSess.Err).To(gbytes.Say("ERROR: Invalid arguments, usage: .*extract.exe <rootfs-tarball> <output-dir>"))
		})
	})
})

func getCreatedTime(file string) int64 {
	fi, err := os.Stat(file)
	Expect(err).NotTo(HaveOccurred())
	return fi.Sys().(*syscall.Win32FileAttributeData).CreationTime.Nanoseconds()
}
