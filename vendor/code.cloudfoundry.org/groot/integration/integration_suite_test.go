package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var tootBinPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build("code.cloudfoundry.org/groot/integration/cmd/toot")
	Expect(err).NotTo(HaveOccurred())
	return []byte(binPath)
}, func(tootBinPathBytes []byte) {
	tootBinPath = string(tootBinPathBytes)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
