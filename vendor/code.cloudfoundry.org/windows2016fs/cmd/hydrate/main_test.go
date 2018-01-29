package main_test

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/archiver/extractor"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Hydrate", func() {
	var (
		outputDir        string
		hydrateArgs      []string
		imageName        string
		imageTag         string
		imageTarballName string
		imageContentsDir string
	)

	BeforeEach(func() {
		var err error
		outputDir, err = ioutil.TempDir("", "hydrateOutput")
		Expect(err).NotTo(HaveOccurred())

		imageContentsDir, err = ioutil.TempDir("", "image-contents")
		Expect(err).NotTo(HaveOccurred())

		imageName = "pivotalgreenhouse/windows2016fs-hydrate"
		imageTag = "1.0.0"
		nameParts := strings.Split(imageName, "/")
		Expect(len(nameParts)).To(Equal(2))
		imageTarballName = fmt.Sprintf("%s-%s.tgz", nameParts[1], imageTag)

		hydrateArgs = []string{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(outputDir)).To(Succeed())
		Expect(os.RemoveAll(imageContentsDir)).To(Succeed())
	})

	Context("when provided an output directory", func() {
		Context("when provided an image tag", func() {
			BeforeEach(func() {
				hydrateArgs = []string{"--outputDir", outputDir, "--image", imageName, "--tag", imageTag}
			})

			It("creates a valid oci-layout file", func() {
				hydrateSess := runHydrate(hydrateArgs)
				Eventually(hydrateSess).Should(gexec.Exit(0))

				tarball := filepath.Join(outputDir, imageTarballName)
				extractTarball(tarball, imageContentsDir)

				ociLayoutFile := filepath.Join(imageContentsDir, "oci-layout")

				var il v1.ImageLayout
				content, err := ioutil.ReadFile(ociLayoutFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(json.Unmarshal(content, &il)).To(Succeed())
				Expect(il.Version).To(Equal(specs.Version))
			})

			It("downloads all the layers with the required metadata files", func() {
				hydrateSess := runHydrate(hydrateArgs)
				Eventually(hydrateSess).Should(gexec.Exit(0))

				tarball := filepath.Join(outputDir, imageTarballName)
				extractTarball(tarball, imageContentsDir)

				im := loadManifest(imageContentsDir)
				ic := loadConfig(imageContentsDir)

				for i, layer := range im.Layers {
					Expect(layer.MediaType).To(Equal(v1.MediaTypeImageLayerGzip))

					layerFile := filename(imageContentsDir, layer)
					fi, err := os.Stat(layerFile)
					Expect(err).NotTo(HaveOccurred())
					Expect(fi.Size()).To(Equal(layer.Size))

					Expect(sha256Sum(layerFile)).To(Equal(layer.Digest.Encoded()))

					Expect(diffID(layerFile)).To(Equal(ic.RootFS.DiffIDs[i].Encoded()))
				}
			})

			Describe("tarball sha", func() {
				It("creates an identical tarball when run multiple times", func() {
					hydrateSess := runHydrate([]string{"--outputDir", outputDir, "--image", imageName, "--tag", imageTag})
					Eventually(hydrateSess).Should(gexec.Exit(0))
					tarballPath := filepath.Join(outputDir, imageTarballName)
					actualSha1 := sha256Sum(tarballPath)
					Expect(os.Remove(tarballPath)).To(Succeed())

					hydrateSess = runHydrate([]string{"--outputDir", outputDir, "--image", imageName, "--tag", imageTag})
					Eventually(hydrateSess).Should(gexec.Exit(0))
					actualSha2 := sha256Sum(filepath.Join(outputDir, imageTarballName))

					Expect(actualSha1).To(Equal(actualSha2))
				})
			})

			Context("when not provided an image tag", func() {
				BeforeEach(func() {
					imageTag = "latest"
					nameParts := strings.Split(imageName, "/")
					Expect(len(nameParts)).To(Equal(2))
					imageTarballName = fmt.Sprintf("%s-%s.tgz", nameParts[1], imageTag)
					hydrateArgs = []string{"--outputDir", outputDir, "--image", imageName}
				})

				It("downloads the latest image version", func() {
					hydrateSess := runHydrate(hydrateArgs)
					Eventually(hydrateSess).Should(gexec.Exit(0))

					tarball := filepath.Join(outputDir, imageTarballName)
					extractTarball(tarball, imageContentsDir)

					im := loadManifest(imageContentsDir)
					ic := loadConfig(imageContentsDir)

					for i, layer := range im.Layers {
						Expect(layer.MediaType).To(Equal(v1.MediaTypeImageLayerGzip))

						layerFile := filename(imageContentsDir, layer)
						fi, err := os.Stat(layerFile)
						Expect(err).NotTo(HaveOccurred())
						Expect(fi.Size()).To(Equal(layer.Size))

						Expect(sha256Sum(layerFile)).To(Equal(layer.Digest.Encoded()))

						Expect(layer.MediaType).To(Equal(v1.MediaTypeImageLayerGzip))
						Expect(diffID(layerFile)).To(Equal(ic.RootFS.DiffIDs[i].Encoded()))
					}
				})
			})
		})

		Context("when not provided an image", func() {
			BeforeEach(func() {
				hydrateArgs = []string{"--outputDir", outputDir}
			})

			It("errors", func() {
				hydrateSess := runHydrate(hydrateArgs)
				Eventually(hydrateSess).Should(gexec.Exit())
				Expect(hydrateSess.ExitCode()).ToNot(Equal(0))
				Expect(string(hydrateSess.Err.Contents())).To(ContainSubstring("ERROR: No image name provided"))
			})
		})
	})

	Context("when the output directory does not exist", func() {
		BeforeEach(func() {
			hydrateArgs = []string{"--image", imageName, "--tag", imageTag, "--outputDir", filepath.Join(outputDir, "random-dir")}
		})

		It("creates it and outputs the image tarball to that directory", func() {
			hydrateSess := runHydrate(hydrateArgs)
			Eventually(hydrateSess).Should(gexec.Exit(0))
			Expect(filepath.Join(outputDir, "random-dir", imageTarballName)).To(BeAnExistingFile())
		})
	})

	Context("when no output directory is provided", func() {
		BeforeEach(func() {
			hydrateArgs = []string{"--image", imageName, "--tag", imageTag}
			Expect(os.RemoveAll(filepath.Join(os.TempDir(), imageTarballName))).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(filepath.Join(os.TempDir(), imageTarballName))).To(Succeed())
		})

		It("outputs to the system temp directory", func() {
			hydrateSess := runHydrate(hydrateArgs)
			Eventually(hydrateSess).Should(gexec.Exit(0))
			Expect(filepath.Join(os.TempDir(), imageTarballName)).To(BeAnExistingFile())
		})
	})
})

func extractTarball(path string, outputDir string) {
	err := extractor.NewTgz().Extract(path, outputDir)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

func sha256Sum(file string) string {
	Expect(file).To(BeAnExistingFile())
	f, err := os.Open(file)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprintf("%x", h.Sum(nil))
}

func diffID(file string) string {
	Expect(file).To(BeAnExistingFile())
	f, err := os.Open(file)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()

	gz, err := gzip.NewReader(f)
	Expect(err).NotTo(HaveOccurred())
	defer gz.Close()

	h := sha256.New()
	_, err = io.Copy(h, gz)
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprintf("%x", h.Sum(nil))
}

func filename(dir string, desc v1.Descriptor) string {
	return filepath.Join(dir, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
}

func loadIndex(outDir string) v1.Index {
	var ii v1.Index
	content, err := ioutil.ReadFile(filepath.Join(outDir, "index.json"))
	Expect(err).NotTo(HaveOccurred())
	Expect(json.Unmarshal(content, &ii)).To(Succeed())

	return ii
}

func loadManifest(outDir string) v1.Manifest {
	ii := loadIndex(outDir)

	content, err := ioutil.ReadFile(filename(outDir, ii.Manifests[0]))
	Expect(err).NotTo(HaveOccurred())

	var im v1.Manifest
	Expect(json.Unmarshal(content, &im)).To(Succeed())
	return im
}

func loadConfig(outDir string) v1.Image {
	im := loadManifest(outDir)

	content, err := ioutil.ReadFile(filename(outDir, im.Config))
	Expect(err).NotTo(HaveOccurred())

	var ic v1.Image
	Expect(json.Unmarshal(content, &ic)).To(Succeed())
	return ic
}

func runHydrate(args []string) *gexec.Session {
	command := exec.Command(hydrateBin, args...)
	hydrateSess, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return hydrateSess
}
