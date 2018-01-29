package main_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestHydrate(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Second * 300)
	RunSpecs(t, "Hydrate Suite")
}

var hydrateBin string

var _ = BeforeSuite(func() {
	var err error
	hydrateBin, err = gexec.Build("code.cloudfoundry.org/windows2016fs/cmd/hydrate")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
