package metadata_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	metadata "code.cloudfoundry.org/windows2016fs/oci-metadata"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Reader", func() {
	var (
		srcDir   string
		manifest v1.Manifest
		config   v1.Image
		index    v1.Index
		diffIds  []digest.Digest
		layers   []v1.Descriptor
		r        *metadata.Reader
	)

	const (
		layer1 = "some-gzipped-data"
		layer2 = "more-gzipped"
		layer3 = "another-layer"

		layer1diffId = "dddddd"
		layer2diffId = "eeeeee"
		layer3diffId = "ffffff"
	)

	BeforeEach(func() {
		var err error
		srcDir, err = ioutil.TempDir("", "windows2016fs.metadata.reader")
		Expect(err).NotTo(HaveOccurred())

		diffIds = []digest.Digest{
			digest.NewDigestFromEncoded("sha256", layer1diffId),
			digest.NewDigestFromEncoded("sha256", layer2diffId),
			digest.NewDigestFromEncoded("sha256", layer3diffId),
		}

		config = v1.Image{
			Architecture: "amd64",
			OS:           "windows",
			RootFS:       v1.RootFS{Type: "layers", DiffIDs: diffIds},
		}
		cdesc := writeBlob(srcDir, config)
		cdesc.MediaType = v1.MediaTypeImageConfig

		layers = []v1.Descriptor{
			{Digest: writeLayer(srcDir, layer1), MediaType: v1.MediaTypeImageLayerGzip},
			{Digest: writeLayer(srcDir, layer2), MediaType: v1.MediaTypeImageLayerGzip},
			{Digest: writeLayer(srcDir, layer3), MediaType: v1.MediaTypeImageLayerGzip},
		}

		manifest = v1.Manifest{
			Config: cdesc,
			Layers: layers,
		}
		mdesc := writeBlob(srcDir, manifest)
		mdesc.MediaType = v1.MediaTypeImageManifest

		index = v1.Index{
			Manifests: []v1.Descriptor{mdesc},
		}

		writeIndex(srcDir, index)
		r = metadata.NewReader(srcDir)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(srcDir)).To(Succeed())
	})

	It("loads the manifest and config from disk", func() {
		m, c, err := r.Read()
		Expect(err).To(Succeed())

		Expect(m).To(Equal(manifest))
		Expect(c).To(Equal(config))
	})

	Context("# manifests in index.json is not 1", func() {
		BeforeEach(func() {
			index = v1.Index{
				Manifests: []v1.Descriptor{
					{Digest: digest.Digest("first manifest")},
					{Digest: digest.Digest("another manifest")},
				},
			}

			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid # of manifests: expected 1, found 2"))
		})
	})

	Context("manifest doesn't match sha256", func() {
		var (
			originalSha string
			newSha      string
		)

		BeforeEach(func() {
			originalSha = index.Manifests[0].Digest.Encoded()

			manifestFile := filepath.Join(srcDir, "blobs", "sha256", originalSha)
			manifestData := []byte(`{"config":{},"layers":[]}`)

			newSha = fmt.Sprintf("%x", sha256.Sum256(manifestData))

			Expect(ioutil.WriteFile(manifestFile, manifestData, 0644)).To(Succeed())
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("sha256 mismatch: expected %s, found %s", originalSha, newSha)))
		})
	})

	Context("manifest in index.json doesn't have os windows", func() {
		BeforeEach(func() {
			index.Manifests[0].Platform = &v1.Platform{OS: "linux", Architecture: "amd64"}
			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid platform: expected windows/amd64, found linux/amd64"))
		})
	})

	Context("manifest in index.json doesn't have arch amd64", func() {
		BeforeEach(func() {
			index.Manifests[0].Platform = &v1.Platform{OS: "windows", Architecture: "some-cpu"}
			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid platform: expected windows/amd64, found windows/some-cpu"))
		})
	})

	Context("manifest in index.json doesn't have os windows or arch amd64", func() {
		BeforeEach(func() {
			index.Manifests[0].Platform = &v1.Platform{OS: "linux", Architecture: "some-cpu"}
			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid platform: expected windows/amd64, found linux/some-cpu"))
		})
	})

	Context("manifest in index.json has wrong media type", func() {
		BeforeEach(func() {
			index.Manifests[0].MediaType = "not-a-manifest"
			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wrong media type for manifest: not-a-manifest"))
		})
	})

	Context("config in manifest has wrong media type", func() {
		BeforeEach(func() {
			manifest.Config.MediaType = "not-a-config"

			mdesc := writeBlob(srcDir, manifest)
			mdesc.MediaType = v1.MediaTypeImageManifest

			index = v1.Index{
				Manifests: []v1.Descriptor{mdesc},
			}

			writeIndex(srcDir, index)
			r = metadata.NewReader(srcDir)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("wrong media type for image config: not-a-config"))
		})
	})

	Context("config doesn't match sha256", func() {
		var (
			originalSha string
			newSha      string
		)

		BeforeEach(func() {
			originalSha = manifest.Config.Digest.Encoded()

			configFile := filepath.Join(srcDir, "blobs", "sha256", originalSha)
			configData := []byte(`{"rootfs":{}}`)

			newSha = fmt.Sprintf("%x", sha256.Sum256(configData))

			Expect(ioutil.WriteFile(configFile, configData, 0644)).To(Succeed())
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("sha256 mismatch: expected %s, found %s", originalSha, newSha)))
		})
	})

	Context("config doesn't have os/arch windows/amd64", func() {
		BeforeEach(func() {
			config.Architecture = "cpu-3"
			config.OS = "windows"

			cdesc := writeBlob(srcDir, config)
			cdesc.MediaType = v1.MediaTypeImageConfig

			manifest = v1.Manifest{
				Config: cdesc,
			}
			mdesc := writeBlob(srcDir, manifest)
			mdesc.MediaType = v1.MediaTypeImageManifest

			index = v1.Index{
				Manifests: []v1.Descriptor{mdesc},
			}

			writeIndex(srcDir, index)
			r = metadata.NewReader(srcDir)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid platform: expected windows/amd64, found windows/cpu-3"))
		})
	})

	Context("rootfs type is not layers", func() {
		BeforeEach(func() {
			config.RootFS = v1.RootFS{
				Type: "something-else",
			}

			cdesc := writeBlob(srcDir, config)
			cdesc.MediaType = v1.MediaTypeImageConfig

			manifest = v1.Manifest{
				Config: cdesc,
			}
			mdesc := writeBlob(srcDir, manifest)
			mdesc.MediaType = v1.MediaTypeImageManifest

			index = v1.Index{
				Manifests: []v1.Descriptor{mdesc},
			}

			writeIndex(srcDir, index)
			r = metadata.NewReader(srcDir)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid rootfs type: something-else"))
		})
	})

	Context("layers are not all correct media type", func() {
		BeforeEach(func() {
			layers = []v1.Descriptor{
				{Digest: writeLayer(srcDir, layer1), MediaType: "not-a-tar.gz"},
				{Digest: writeLayer(srcDir, layer2), MediaType: "not-a-tar.gz"},
				{Digest: writeLayer(srcDir, layer3), MediaType: "not-a-tar.gz"},
			}

			manifest.Layers = layers

			mdesc := writeBlob(srcDir, manifest)
			mdesc.MediaType = v1.MediaTypeImageManifest

			index = v1.Index{
				Manifests: []v1.Descriptor{mdesc},
			}

			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid layer media type: not-a-tar.gz"))
		})
	})

	Context("layers do not match sha256", func() {
		var (
			originalSha string
			newSha      string
		)

		BeforeEach(func() {
			originalSha = layers[0].Digest.Encoded()
			layerFile := filepath.Join(srcDir, "blobs", "sha256", originalSha)

			newContents := []byte("a-different-layer")
			newSha = fmt.Sprintf("%x", sha256.Sum256(newContents))
			Expect(ioutil.WriteFile(layerFile, newContents, 0644)).To(Succeed())
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("invalid layer: sha256 mismatch: expected %s, found %s", originalSha, newSha)))
		})
	})

	Context("number of layers and number of diffids do not match", func() {
		BeforeEach(func() {
			layers = []v1.Descriptor{
				{Digest: writeLayer(srcDir, layer1), MediaType: v1.MediaTypeImageLayerGzip},
				{Digest: writeLayer(srcDir, layer2), MediaType: v1.MediaTypeImageLayerGzip},
			}
			manifest.Layers = layers

			mdesc := writeBlob(srcDir, manifest)
			mdesc.MediaType = v1.MediaTypeImageManifest

			index = v1.Index{
				Manifests: []v1.Descriptor{mdesc},
			}

			writeIndex(srcDir, index)
		})

		It("returns a descriptive error", func() {
			_, _, err := r.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("manifest + config mismatch: 2 layers, 3 diffIDs"))
		})
	})
})

func writeLayer(outDir string, contents string) digest.Digest {
	blobSha := fmt.Sprintf("%x", sha256.Sum256([]byte(contents)))

	blobsDir := filepath.Join(outDir, "blobs", "sha256")
	Expect(os.MkdirAll(blobsDir, 0755)).To(Succeed())

	Expect(ioutil.WriteFile(filepath.Join(blobsDir, blobSha), []byte(contents), 0644)).To(Succeed())

	return digest.NewDigestFromEncoded(digest.SHA256, blobSha)
}

func writeBlob(outDir string, blob interface{}) v1.Descriptor {
	data, err := json.Marshal(blob)
	Expect(err).NotTo(HaveOccurred())

	blobSha := fmt.Sprintf("%x", sha256.Sum256(data))

	blobsDir := filepath.Join(outDir, "blobs", "sha256")
	Expect(os.MkdirAll(blobsDir, 0755)).To(Succeed())

	Expect(ioutil.WriteFile(filepath.Join(blobsDir, blobSha), data, 0644)).To(Succeed())

	return v1.Descriptor{
		Size:   int64(len(data)),
		Digest: digest.NewDigestFromEncoded(digest.SHA256, blobSha),
	}
}

func writeConfig(outDir string, diffIds []digest.Digest) v1.Descriptor {
	ic := v1.Image{
		Architecture: "amd64",
		OS:           "windows",
		RootFS:       v1.RootFS{Type: "layers", DiffIDs: diffIds},
	}

	d := writeBlob(outDir, ic)
	d.MediaType = v1.MediaTypeImageConfig
	return d
}

func writeManifest(outDir string, config v1.Descriptor, layers []v1.Descriptor) v1.Descriptor {
	im := v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    config,
		Layers:    layers,
	}

	d := writeBlob(outDir, im)
	d.MediaType = v1.MediaTypeImageManifest
	d.Platform = &v1.Platform{OS: "windows", Architecture: "amd64"}
	return d
}

func writeIndex(outDir string, i v1.Index) {
	data, err := json.Marshal(i)
	Expect(err).NotTo(HaveOccurred())
	Expect(ioutil.WriteFile(filepath.Join(outDir, "index.json"), data, 0644)).To(Succeed())
}
