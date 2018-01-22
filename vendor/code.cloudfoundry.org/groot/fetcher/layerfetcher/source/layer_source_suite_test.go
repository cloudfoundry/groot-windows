package source_test

import (
	"archive/tar"
	"io"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	RegistryUsername string
	RegistryPassword string
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeEach(func() {
		RegistryUsername = os.Getenv("REGISTRY_USERNAME")
		RegistryPassword = os.Getenv("REGISTRY_PASSWORD")
	})

	RunSpecs(t, "Layer Fetcher Source Suite")
}

func tarEntries(tarFile io.Reader) []string {
	tr := tar.NewReader(tarFile)
	entries := []string{}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		Expect(err).NotTo(HaveOccurred())
		entries = append(entries, hdr.Name)
	}

	return entries
}
