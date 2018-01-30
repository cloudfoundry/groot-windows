package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/windows2016fs/hydrator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	grootBin string

	imageTags = []string{
		"regularfile",
		"whiteout",
		"link",
	}
	imageTgzDir string
	keepDir     bool
)

func TestGrootWindows(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Millisecond * 200)
	RunSpecs(t, "GrootWindows Suite")
}

var _ = BeforeSuite(func() {
	var err error

	grootBin, err = gexec.Build("code.cloudfoundry.org/groot-windows")
	Expect(err).ToNot(HaveOccurred())

	imageTgzDir, keepDir = os.LookupEnv("GROOT_WINDOWS_IMAGE_TGZS")

	if !keepDir {
		imageTgzDir, err = ioutil.TempDir("", "groot-windows-image-tgzs")
		Expect(err).ToNot(HaveOccurred())
	}

	for _, tag := range imageTags {
		_, err := os.Stat(filepath.Join(imageTgzDir, fmt.Sprintf("groot-windows-test-%s.tgz", tag)))
		if err != nil && os.IsNotExist(err) {
			Expect(hydrator.New(imageTgzDir, "pivotalgreenhouse/groot-windows-test", tag).Run()).To(Succeed())
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	if !keepDir {
		Expect(os.RemoveAll(imageTgzDir)).To(Succeed())
	}
	gexec.CleanupBuildArtifacts()
})
