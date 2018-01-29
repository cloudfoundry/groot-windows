package integration_test

import (
	"io/ioutil"
	"os"
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

	imageTgzDir, err = ioutil.TempDir("", "groot-windows-image-tgzs")
	Expect(err).ToNot(HaveOccurred())

	for _, tag := range imageTags {
		Expect(hydrator.New(imageTgzDir, "pivotalgreenhouse/groot-windows-test", tag).Run()).To(Succeed())
	}
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll(imageTgzDir)).To(Succeed())
	gexec.CleanupBuildArtifacts()
})
