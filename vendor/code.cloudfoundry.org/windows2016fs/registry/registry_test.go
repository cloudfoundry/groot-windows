package registry_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/windows2016fs/registry"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Registry", func() {
	var (
		r              *registry.Registry
		authServer     *ghttp.Server
		registryServer *ghttp.Server
		outputDir      string
		manifest       v1.Manifest
		imageName      = "some-image-name"
		imageRef       = "some-image-ref"
		token          = "some-token"
	)

	BeforeEach(func() {
		var err error
		authServer = ghttp.NewServer()
		registryServer = ghttp.NewServer()
		r = registry.New(authServer.URL(), registryServer.URL(), imageName, imageRef)

		outputDir, err = ioutil.TempDir("", "hydrate.registry.test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		authServer.Close()
		registryServer.Close()
		Expect(os.RemoveAll(outputDir)).To(Succeed())
	})

	Describe("Manifest", func() {
		Context("successful download", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)
				manifest = v1.Manifest{Config: v1.Descriptor{MediaType: "some-media-type"}}
				marshaledManifest, err := json.Marshal(manifest)
				Expect(err).NotTo(HaveOccurred())
				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/manifests/%s", imageName, imageRef), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.VerifyHeader(http.Header{"Accept": []string{"application/vnd.docker.distribution.manifest.v2+json", "application/vnd.docker.distribution.manifest.list.v2+json"}}),
						ghttp.RespondWith(http.StatusOK, marshaledManifest),
					),
				)
			})

			It("returns a manifest for the given image and ref", func() {
				actualManifest, err := r.Manifest()
				Expect(err).NotTo(HaveOccurred())
				Expect(actualManifest).To(Equal(manifest))
			})
		})

		Context("the registry server returns a non-200 response", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)
				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/manifests/%s", imageName, imageRef), ""),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				_, err := r.Manifest()
				Expect(err).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})

		Context("the auth server returns a non-200 response", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				_, err := r.Manifest()
				Expect(err).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})
	})

	Describe("DownloadLayer", func() {
		var (
			layer     v1.Descriptor
			layerData = "some-layer-data"
			layerSHA  = "a4dce48a216523fad0e7932218c9e5e6d6a4753df784ed2f6ec4e5ac9405e2a5"
		)

		Context("for foreign container storage layer", func() {
			var (
				foreignServer *ghttp.Server
			)
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				foreignServer = ghttp.NewServer()
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip",
					URLs:      []string{foreignServer.URL()},
				}

				foreignServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/", ""),
						ghttp.RespondWith(http.StatusOK, []byte(layerData)),
					),
				)
			})
			AfterEach(func() {
				foreignServer.Close()
			})

			It("downloads a layer for the given image and blob digest", func() {
				Expect(r.DownloadLayer(layer, outputDir)).To(Succeed())

				data, err := ioutil.ReadFile(filepath.Join(outputDir, layerSHA))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(layerData))
			})
		})

		Context("for a docker hosted layer", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				}

				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, layer.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusOK, []byte(layerData)),
					),
				)
			})

			It("downloads a layer for the given image and blob digest", func() {
				Expect(r.DownloadLayer(layer, outputDir)).To(Succeed())

				data, err := ioutil.ReadFile(filepath.Join(outputDir, layerSHA))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(data)).To(Equal(layerData))
			})
		})

		Context("the auth server returns a non-200 response", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				}

				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})

		Context("the sha256 does not match", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				}

				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, layer.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusOK, []byte("some-different-data")),
					),
				)
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.SHAMismatchError{}))
			})
		})

		Context("the digest algorithm is not sha256", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest: digest.NewDigestFromEncoded(digest.SHA384, strings.Repeat("a", 96)),
				}
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.DigestAlgorithmError{}))
			})
		})

		Context("the digest is incorrectly formatted", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest: digest.Digest("not-a-digest"),
				}
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause.Error()).To(Equal("invalid checksum digest format"))
			})
		})

		Context("the registry server returns a non-200 response", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				}

				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, layer.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})

		Context("the media type is invalid", func() {
			BeforeEach(func() {
				layer = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", layerSHA),
					MediaType: "some-invalid-media-type",
				}
			})

			It("returns an error", func() {
				err := r.DownloadLayer(layer, outputDir)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.InvalidMediaTypeError{}))
			})
		})
	})

	Describe("Config", func() {
		var (
			config     v1.Descriptor
			configData = `{"os":"some-os","architecture":"some-arch"}`
			configLen  = int64(len(configData))
			configSHA  = "578ede1ce55a039ccad3151fe2cd96b332c389503416666cc78ab48d3803b2a4"
		)

		BeforeEach(func() {
			config = v1.Descriptor{
				Digest:    digest.NewDigestFromEncoded("sha256", configSHA),
				MediaType: "application/vnd.docker.container.image.v1+json",
				Size:      configLen,
			}
		})

		Context("a successful request", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, config.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusOK, []byte(configData)),
					),
				)
			})

			It("returns the config object for the given descriptor", func() {
				c, err := r.Config(config)
				Expect(err).NotTo(HaveOccurred())

				Expect(c.Architecture).To(Equal("some-arch"))
				Expect(c.OS).To(Equal("some-os"))
			})
		})

		Context("the auth server returns a non-200 response", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})

		Context("the sha256 does not match", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, config.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusOK, []byte("some-different-data")),
					),
				)
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.SHAMismatchError{}))
			})
		})

		Context("the digest algorithm is not sha256", func() {
			BeforeEach(func() {
				config = v1.Descriptor{
					MediaType: "application/vnd.docker.container.image.v1+json",
					Digest:    digest.NewDigestFromEncoded(digest.SHA384, strings.Repeat("a", 96)),
				}
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.DigestAlgorithmError{}))
			})
		})

		Context("the digest is incorrectly formatted", func() {
			BeforeEach(func() {
				config = v1.Descriptor{
					MediaType: "application/vnd.docker.container.image.v1+json",
					Digest:    digest.Digest("not-a-digest"),
				}
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause.Error()).To(Equal("invalid checksum digest format"))
			})
		})

		Context("the registry server returns a non-200 response", func() {
			BeforeEach(func() {
				authServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/token", fmt.Sprintf("service=registry.docker.io&scope=repository:%s:pull", imageName)),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"token": "%s"}`, token)),
					),
				)

				registryServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", fmt.Sprintf("/v2/%s/blobs/%s", imageName, config.Digest), ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"Bearer " + token}}),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.HTTPNotOKError{}))
			})
		})

		Context("the media type is invalid", func() {
			BeforeEach(func() {
				config = v1.Descriptor{
					Digest:    digest.NewDigestFromEncoded("sha256", configSHA),
					MediaType: "some-invalid-media-type",
				}
			})

			It("returns an error", func() {
				_, err := r.Config(config)
				Expect(err).To(BeAssignableToTypeOf(&registry.DownloadError{}))
				Expect(err.(*registry.DownloadError).Cause).To(BeAssignableToTypeOf(&registry.InvalidMediaTypeError{}))
			})
		})
	})
})
