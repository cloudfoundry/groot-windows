package layer_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/windows2016fs/layer"
	"code.cloudfoundry.org/windows2016fs/layer/layerfakes"

	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Layer Manager", func() {
	var (
		lm         *layer.Manager
		w          *layerfakes.FakeWriter
		outputDir  string
		driverInfo hcsshim.DriverInfo
	)
	const layerId = "layer1"

	BeforeEach(func() {
		var err error
		outputDir, err = ioutil.TempDir("", "windows2016fs.layer_test")
		Expect(err).NotTo(HaveOccurred())

		w = &layerfakes.FakeWriter{}
		driverInfo = hcsshim.DriverInfo{HomeDir: outputDir}
		lm = layer.NewManager(driverInfo, w)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(outputDir)).To(Succeed())
	})

	Describe("State", func() {
		Context("the layer directory does not exist", func() {
			It("returns NotExist", func() {
				state, err := lm.State(layerId)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(layer.State(layer.NotExist)))
			})
		})

		Context("the layer directory exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(outputDir, layerId), 0755)).To(Succeed())
			})

			Context("and does not contain a .complete file", func() {
				It("returns Incomplete", func() {
					state, err := lm.State(layerId)
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(layer.State(layer.Incomplete)))
				})
			})

			Context("and contains a .complete file with the wrong id", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(outputDir, layerId, ".complete"), []byte("abcde"), 0644))
				})

				It("returns Incomplete", func() {
					state, err := lm.State(layerId)
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(layer.State(layer.Incomplete)))
				})
			})

			Context("and contains a .complete file with the matching id", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(outputDir, layerId, ".complete"), []byte(layerId), 0644))
				})

				It("returns Valid", func() {
					state, err := lm.State(layerId)
					Expect(err).NotTo(HaveOccurred())
					Expect(state).To(Equal(layer.State(layer.Valid)))
				})
			})
		})
	})

	Describe("Delete", func() {
		var layerDir string

		BeforeEach(func() {
			layerDir = filepath.Join(outputDir, layerId)
			Expect(os.MkdirAll(layerDir, 0755)).To(Succeed())
		})

		It("removes the directory", func() {
			Expect(layerDir).To(BeADirectory())
			Expect(lm.Delete(layerId)).To(Succeed())
			Expect(layerDir).NotTo(BeADirectory())
		})
	})

	Describe("Extract", func() {
		var parentLayerIds []string

		const layerGzipFile = "path-to-tar.gz"

		BeforeEach(func() {
			parentLayerIds = []string{"layer5", "layer6"}
		})

		It("extracts the provided layer tgz usin a layer writer", func() {
			Expect(lm.Extract(layerGzipFile, layerId, parentLayerIds)).To(Succeed())

			Expect(w.SetHCSLayerWriterCallCount()).To(Equal(1))
			di, id, paths := w.SetHCSLayerWriterArgsForCall(0)
			Expect(di).To(Equal(driverInfo))
			Expect(id).To(Equal(layerId))
			Expect(paths).To(Equal([]string{filepath.Join(outputDir, "layer5"), filepath.Join(outputDir, "layer6")}))

			Expect(w.WriteLayerCallCount()).To(Equal(1))
			Expect(w.WriteLayerArgsForCall(0)).To(Equal(layerGzipFile))
		})

		It("it creates the directory to extract the layer to", func() {
			Expect(filepath.Join(outputDir, layerId)).NotTo(BeADirectory())
			Expect(lm.Extract(layerGzipFile, layerId, parentLayerIds)).To(Succeed())
			Expect(filepath.Join(outputDir, layerId)).To(BeADirectory())
		})

		It("it writes a layerchain.json to the output directory", func() {
			Expect(lm.Extract(layerGzipFile, layerId, parentLayerIds)).To(Succeed())
			data, err := ioutil.ReadFile(filepath.Join(outputDir, layerId, "layerchain.json"))
			Expect(err).NotTo(HaveOccurred())

			paths := []string{}
			Expect(json.Unmarshal(data, &paths)).To(Succeed())
			Expect(paths).To(Equal([]string{filepath.Join(outputDir, "layer5"), filepath.Join(outputDir, "layer6")}))
		})

		Context("when the layer has no parents", func() {
			BeforeEach(func() {
				parentLayerIds = []string{}
			})

			It("it does not write a layerchain.json to the output directory", func() {
				Expect(lm.Extract(layerGzipFile, layerId, parentLayerIds)).To(Succeed())
				Expect(filepath.Join(outputDir, layerId, "layerchain.json")).NotTo(BeAnExistingFile())
			})
		})

		It("writes a .complete file containing the layerId to the output directory", func() {
			Expect(lm.Extract(layerGzipFile, layerId, parentLayerIds)).To(Succeed())
			data, err := ioutil.ReadFile(filepath.Join(outputDir, layerId, ".complete"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(layerId))
		})
	})
})
