package compress_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCompress(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compress Suite")
}
