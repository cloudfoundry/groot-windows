package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestExtract(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Second * 300)
	RunSpecs(t, "Extract Suite")
}

var (
	hydrateBin string
	extractBin string
	rootfsTgz  string
	tempDir    string
)

var _ = BeforeSuite(func() {
	var err error
	hydrateBin, err = gexec.Build("code.cloudfoundry.org/windows2016fs/cmd/hydrate")
	Expect(err).NotTo(HaveOccurred())

	extractBin, err = gexec.Build("code.cloudfoundry.org/windows2016fs/cmd/extract")
	Expect(err).NotTo(HaveOccurred())

	imageName := "pivotalgreenhouse/windows2016fs-hydrate"
	imageTag := "2.0.0"
	imageTarballName := "windows2016fs-hydrate-2.0.0.tgz"

	tempDir, err = ioutil.TempDir("", "extract.hydrated.test")
	Expect(err).NotTo(HaveOccurred())

	rootfsTgz = filepath.Join(tempDir, imageTarballName)

	cmd := exec.Command(hydrateBin, "--outputDir", tempDir, "--image", imageName, "--tag", imageTag)
	hydrateSess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(hydrateSess).Should(gexec.Exit(0))
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	Expect(os.RemoveAll(tempDir)).To(Succeed())
})
