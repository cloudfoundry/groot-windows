package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"code.cloudfoundry.org/groot/integration/cmd/toot/toot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("groot", func() {
	Describe("create", func() {
		var (
			rootfsURI            string
			handle               = "some-handle"
			logLevel             string
			configFilePath       string
			env                  []string
			tempDir              string
			stdout               *bytes.Buffer
			stderr               *bytes.Buffer
			notFoundRuntimeError = map[string]string{
				"linux":   "no such file or directory",
				"windows": "The system cannot find the file specified.",
			}
		)

		argFilePath := func(filename string) string {
			return filepath.Join(tempDir, filename)
		}

		readTestArgsFile := func(filename string, ptr interface{}) {
			content, err := ioutil.ReadFile(argFilePath(filename))
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(content, ptr)).To(Succeed())
		}

		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "groot-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			configFilePath = filepath.Join(tempDir, "groot-config.yml")
			rootfsURI = filepath.Join(tempDir, "rootfs.tar")

			logLevel = ""
			env = []string{"TOOT_BASE_DIR=" + tempDir}
			stdout = new(bytes.Buffer)
			stderr = new(bytes.Buffer)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		runTootCmd := func() error {
			tootArgv := []string{"--config", configFilePath, "create", rootfsURI, handle}
			tootCmd := exec.Command(tootBinPath, tootArgv...)
			tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
			tootCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
			tootCmd.Env = append(os.Environ(), env...)
			return tootCmd.Run()
		}

		whenCreationSucceeds := func() {
			It("calls driver.Unpack() with the expected args", func() {
				var args toot.UnpackCalls
				readTestArgsFile(toot.UnpackArgsFileName, &args)
				Expect(args[0].ID).NotTo(BeEmpty())
				Expect(args[0].ParentIDs).To(BeEmpty())
			})

			It("calls driver.Bundle() with expected args", func() {
				var unpackArgs toot.UnpackCalls
				readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)

				var bundleArgs toot.BundleCalls
				readTestArgsFile(toot.BundleArgsFileName, &bundleArgs)
				unpackLayerIds := []string{}
				for _, call := range unpackArgs {
					unpackLayerIds = append(unpackLayerIds, call.ID)
				}
				Expect(bundleArgs[0].ID).To(Equal(handle))
				Expect(bundleArgs[0].LayerIDs).To(ConsistOf(unpackLayerIds))
			})

			It("logs to stderr with an appropriate lager session, defaulting to info level", func() {
				Expect(stderr.String()).To(ContainSubstring("groot.create.bundle.bundle-info"))
				Expect(stderr.String()).NotTo(ContainSubstring("bundle-debug"))
			})

			Context("when no config file is provided", func() {
				BeforeEach(func() {
					configFilePath = ""
				})

				It("uses the default log level", func() {
					Expect(stderr.String()).ToNot(ContainSubstring("bundle-debug"))
					Expect(stderr.String()).To(ContainSubstring("bundle-info"))
				})
			})

			Context("when the log level is specified", func() {
				BeforeEach(func() {
					logLevel = "debug"
				})

				It("logs to stderr with the specified lager level", func() {
					Expect(stderr.String()).To(ContainSubstring("bundle-debug"))
				})
			})

			Describe("subsequent invocations", func() {
				Context("when the rootfs file has not changed", func() {
					It("generates the same layer ID", func() {
						var unpackArgs toot.UnpackCalls
						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						Expect(runTootCmd()).To(Succeed())

						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						secondInvocationLayerID := unpackArgs[0].ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})
			})

			Describe("layer caching", func() {
				It("calls exists", func() {
					var existsArgs toot.ExistsCalls
					readTestArgsFile(toot.ExistsArgsFileName, &existsArgs)
					Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
				})

				Context("when the layer is not cached", func() {
					It("calls unpack with the same layerID", func() {
						var existsArgs toot.ExistsCalls
						readTestArgsFile(toot.ExistsArgsFileName, &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

						Expect(argFilePath(toot.UnpackArgsFileName)).To(BeAnExistingFile())

						var unpackArgs toot.UnpackCalls
						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

						lastCall := len(unpackArgs) - 1
						for i := range unpackArgs {
							Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
						}
					})
				})

				Context("when the layer is cached", func() {
					BeforeEach(func() {
						env = append(env, "TOOT_LAYER_EXISTS=true")
					})

					It("doesn't call unpack", func() {
						Expect(argFilePath(toot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
					})
				})
			})
		}

		whenCreationFails := func() {
			var writeConfigFile bool

			BeforeEach(func() {
				writeConfigFile = true
			})

			JustBeforeEach(func() {
				if writeConfigFile {
					configYml := fmt.Sprintf(`log_level: %s`, logLevel)
					Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())
				}

				tootArgv := []string{"--config", configFilePath, "create", rootfsURI, handle}
				tootCmd := exec.Command(tootBinPath, tootArgv...)
				tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
				tootCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
				tootCmd.Env = append(os.Environ(), env...)
				exitErr := tootCmd.Run()
				Expect(exitErr).To(HaveOccurred())
				Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
			})

			Context("when driver.Bundle() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "TOOT_BUNDLE_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(ContainSubstring("bundle-err\n"))
				})
			})

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "TOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(ContainSubstring("unpack-err\n"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					writeConfigFile = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeConfigFile = false
					Expect(ioutil.WriteFile(configFilePath, []byte("%haha"), 0600)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					logLevel = "lol"
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("lol"))
				})
			})
		}

		Describe("success", func() {
			JustBeforeEach(func() {
				if configFilePath != "" {
					configYml := fmt.Sprintf(`log_level: %s`, logLevel)
					Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())
				}
			})

			Describe("Local images", func() {
				JustBeforeEach(func() {
					Expect(ioutil.WriteFile(rootfsURI, []byte("a-rootfs"), 0600)).To(Succeed())

					Expect(runTootCmd()).To(Succeed())
				})

				whenCreationSucceeds()

				It("calls driver.Unpack() with the correct stream", func() {
					var args toot.UnpackCalls
					readTestArgsFile(toot.UnpackArgsFileName, &args)
					Expect(string(args[0].LayerTarContents)).To(Equal("a-rootfs"))
				})

				Describe("subsequent invocations", func() {
					Context("when the rootfs file timestamp has changed", func() {
						It("generates a different layer ID", func() {
							var unpackArgs toot.UnpackCalls
							readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
							firstInvocationLayerID := unpackArgs[0].ID

							now := time.Now()
							Expect(os.Chtimes(rootfsURI, now.Add(time.Hour), now.Add(time.Hour))).To(Succeed())

							Expect(runTootCmd()).To(Succeed())

							readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
							secondInvocationLayerID := unpackArgs[1].ID

							Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
						})
					})
				})
			})

			Describe("Remote images", func() {
				JustBeforeEach(func() {
					rootfsURI = "docker:///cfgarden/three-layers"

					Expect(runTootCmd()).To(Succeed())
				})

				whenCreationSucceeds()

				Context("when the image has multiple layers", func() {
					It("correctly passes parent IDs to each driver.Unpack() call", func() {
						var args toot.UnpackCalls
						readTestArgsFile(toot.UnpackArgsFileName, &args)

						chainIDs := []string{}
						for _, a := range args {
							Expect(a.ParentIDs).To(Equal(chainIDs))
							chainIDs = append(chainIDs, a.ID)
						}
					})
				})
			})
		})

		Describe("failure", func() {
			Describe("Local Images", func() {
				var createRootfsTar bool

				BeforeEach(func() {
					createRootfsTar = true
				})

				JustBeforeEach(func() {
					if createRootfsTar {
						Expect(ioutil.WriteFile(rootfsURI, []byte("a-rootfs"), 0600)).To(Succeed())
					}
				})

				whenCreationFails()

				Context("when the rootfs URI is not a file", func() {
					BeforeEach(func() {
						createRootfsTar = false
					})

					It("prints an error", func() {
						Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
					})
				})
			})

			Describe("Remote Images", func() {
				BeforeEach(func() {
					rootfsURI = "docker:///cfgarden/three-layers"
				})

				whenCreationFails()
			})

		})
	})
})
