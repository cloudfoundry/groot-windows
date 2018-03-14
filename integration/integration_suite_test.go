package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/hydrator/hydrator"
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

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	grootBin, err = gexec.Build("code.cloudfoundry.org/groot-windows")
	Expect(err).ToNot(HaveOccurred())

	grootDir := filepath.Dir(grootBin)

	o, err := exec.Command("gcc.exe", "-c", "..\\volume\\quota\\quota.c", "-o", filepath.Join(grootDir, "quota.o")).CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(o))

	o, err = exec.Command("gcc.exe",
		"-shared",
		"-o", filepath.Join(grootDir, "quota.dll"),
		filepath.Join(grootDir, "quota.o"),
		"-lole32", "-loleaut32").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), string(o))

	imageTgzDir, keepDir = os.LookupEnv("GROOT_WINDOWS_IMAGE_TGZS")

	if !keepDir {
		imageTgzDir, err = ioutil.TempDir("", "groot-windows-image-tgzs")
		Expect(err).ToNot(HaveOccurred())
	}

	for _, tag := range imageTags {
		_, err := os.Stat(filepath.Join(imageTgzDir, fmt.Sprintf("groot-windows-test-%s.tgz", tag)))
		if err != nil && os.IsNotExist(err) {
			Expect(hydrator.New(imageTgzDir, "cloudfoundry/groot-windows-test", tag, false).Run()).To(Succeed())
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
	}

	testData := make(map[string]string)
	testData["groot_bin"] = grootBin
	testData["image_tgz_dir"] = imageTgzDir
	json, err := json.Marshal(testData)
	Expect(err).NotTo(HaveOccurred())

	return json
}, func(jsonBytes []byte) {
	testData := make(map[string]string)
	Expect(json.Unmarshal(jsonBytes, &testData)).To(Succeed())

	grootBin = testData["groot_bin"]
	imageTgzDir = testData["image_tgz_dir"]
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	if !keepDir {
		Expect(os.RemoveAll(imageTgzDir)).To(Succeed())
	}
	gexec.CleanupBuildArtifacts()
})
