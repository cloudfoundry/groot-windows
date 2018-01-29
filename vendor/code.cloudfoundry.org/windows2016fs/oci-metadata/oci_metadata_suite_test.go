package metadata_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOciMetadata(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OciMetadata Suite")
}
