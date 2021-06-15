package groot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"

	"code.cloudfoundry.org/groot/fetcher/filefetcher"
	"code.cloudfoundry.org/groot/fetcher/layerfetcher"
	"code.cloudfoundry.org/groot/fetcher/layerfetcher/source"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
	"github.com/containers/image/v5/types"
	runspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/urfave/cli"
)

type DiskUsage struct {
	TotalBytesUsed     int64 `json:"total_bytes_used"`
	ExclusiveBytesUsed int64 `json:"exclusive_bytes_used"`
}

type VolumeStats struct {
	DiskUsage DiskUsage `json:"disk_usage"`
}

type ImageMetadata struct {
	Size int64 `json:"size"`
}

type ImageDriver interface {
	Bundle(logger lager.Logger, bundleID string, layerIDs []string, diskLimit int64) (runspec.Spec, error)
	Delete(logger lager.Logger, bundleID string) error
	Stats(logger lager.Logger, bundleID string) (VolumeStats, error)
	WriteMetadata(logger lager.Logger, bundleID string, imageMetadata ImageMetadata) error
}

type VolumeDriver interface {
	Unpack(logger lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) (int64, error)
}

// Driver should implement the filesystem interaction
//go:generate counterfeiter . Driver
type Driver interface {
	ImageDriver
	VolumeDriver
}

// ImagePuller should be able to download and store a remote (or local) image
// and return all its layer information so that it can be bundled together by
// the driver
//go:generate counterfeiter . ImagePuller
type ImagePuller interface {
	Pull(logger lager.Logger, spec imagepuller.ImageSpec) (imagepuller.Image, error)
}

type Groot struct {
	Driver      Driver
	Logger      lager.Logger
	ImagePuller ImagePuller
}

type DockerConfig struct {
	InsecureRegistries []string
	Username           string
	Password           string
}

func Run(driver Driver, argv []string, driverFlags []cli.Flag, version string) {
	// The `Before` closure sets this. This is ugly, but we don't know the log
	// level until the CLI framework has parsed the flags.
	var g *Groot
	var err error
	var fetcher imagepuller.Fetcher
	var conf config

	app := cli.NewApp()
	app.Version = version
	app.Usage = "A garden image plugin"
	app.Flags = append([]cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: "",
			Usage: "Path to config file",
		},
	}, driverFlags...)
	app.Commands = []cli.Command{
		{
			Name: "create",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  "disk-limit-size-bytes",
					Usage: "Inclusive disk limit (i.e: includes all layers in the filesystem)",
				},
				cli.BoolFlag{
					Name:  "exclude-image-from-quota",
					Usage: "Set disk limit to be exclusive (i.e.: excluding image layers)",
				},
				cli.StringFlag{
					Name:  "username",
					Usage: "Username to authenticate in image registry",
				},
				cli.StringFlag{
					Name:  "password",
					Usage: "Password to authenticate in image registry",
				},
			},
			Action: func(ctx *cli.Context) error {
				dockerConfig := DockerConfig{
					InsecureRegistries: conf.InsecureRegistries,
					Username:           ctx.String("username"),
					Password:           ctx.String("password"),
				}

				if err := validateArgs(ctx, 2); err != nil {
					return err
				}

				if fetcher, err = createFetcher(ctx.Args()[0], ctx.Bool("exclude-image-from-quota"), ctx.Int64("disk-limit-size-bytes"), dockerConfig); err != nil {
					return err
				}
				defer fetcher.Close()
				g.ImagePuller = imagepuller.NewImagePuller(fetcher, driver)

				handle := ctx.Args()[1]
				var runtimeSpec runspec.Spec
				runtimeSpec, err = g.Create(handle, ctx.Int64("disk-limit-size-bytes"), ctx.Bool("exclude-image-from-quota"))
				if err != nil {
					return err
				}

				return json.NewEncoder(os.Stdout).Encode(runtimeSpec)
			},
		},
		{
			Name: "pull",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "username",
					Usage: "Username to authenticate in image registry",
				},
				cli.StringFlag{
					Name:  "password",
					Usage: "Password to authenticate in image registry",
				},
			},
			Action: func(ctx *cli.Context) error {
				dockerConfig := DockerConfig{
					InsecureRegistries: conf.InsecureRegistries,
					Username:           ctx.String("username"),
					Password:           ctx.String("password"),
				}
				if err := validateArgs(ctx, 1); err != nil {
					return err
				}

				if fetcher, err = createFetcher(ctx.Args()[0], ctx.Bool("exclude-image-from-quota"), ctx.Int64("disk-limit-size-bytes"), dockerConfig); err != nil {
					return err
				}
				defer fetcher.Close()
				g.ImagePuller = imagepuller.NewImagePuller(fetcher, driver)
				return g.Pull()
			},
		},
		{
			Name: "delete",
			Action: func(ctx *cli.Context) error {
				if err := validateArgs(ctx, 1); err != nil {
					return err
				}
				handle := ctx.Args()[0]
				return g.Delete(handle)
			},
		},
		{
			Name: "stats",
			Action: func(ctx *cli.Context) error {
				if err := validateArgs(ctx, 1); err != nil {
					return err
				}
				handle := ctx.Args()[0]
				stats, err := g.Stats(handle)
				if err != nil {
					return err
				}
				return json.NewEncoder(os.Stdout).Encode(stats)
			},
		},
	}
	app.Before = func(ctx *cli.Context) error {
		var err error
		conf, err = parseConfig(ctx.GlobalString("config"))
		if err != nil {
			return silentError(err)
		}

		logger, err := newLogger(conf.LogLevel)
		if err != nil {
			return err
		}

		g = &Groot{
			Driver: driver,
			Logger: logger,
		}
		return nil
	}

	if err := app.Run(argv); err != nil {
		if _, ok := err.(SilentError); !ok {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func createFetcher(urlAsString string, excludeImageFromQuota bool, diskLimitSizeBytes int64, dockerConfig DockerConfig) (imagepuller.Fetcher, error) {
	imageURL, err := url.Parse(urlAsString)
	if err != nil {
		return nil, err
	}

	if imageURL.Scheme == "oci" || imageURL.Scheme == "docker" {
		systemContext := types.SystemContext{}

		if imageURL.Scheme == "docker" {
			systemContext.DockerInsecureSkipTLSVerify = types.NewOptionalBool(skipTLSValidation(imageURL, dockerConfig.InsecureRegistries))
			systemContext.DockerAuthConfig = &types.DockerAuthConfig{
				Username: dockerConfig.Username,
				Password: dockerConfig.Password,
			}
		}

		layerSource := source.NewLayerSource(systemContext, false, shouldSkipImageQuotaValidation(excludeImageFromQuota, diskLimitSizeBytes), diskLimitSizeBytes, imageURL)

		return layerfetcher.NewLayerFetcher(&layerSource), nil
	}

	return filefetcher.NewFileFetcher(imageURL), nil
}

func shouldSkipImageQuotaValidation(excludeImageFromQuota bool, diskLimitSizeBytes int64) bool {
	return excludeImageFromQuota || diskLimitSizeBytes == 0
}

func newLogger(logLevelStr string) (lager.Logger, error) {
	logLevels := map[string]lager.LogLevel{
		"debug": lager.DEBUG,
		"info":  lager.INFO,
		"error": lager.ERROR,
		"fatal": lager.FATAL,
	}

	logLevel, ok := logLevels[logLevelStr]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", logLevelStr)
	}

	logger := lager.NewLogger("groot")
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, logLevel))

	return logger, nil
}

// SilentError silences errors. urfave/cli already prints certain errors, we
// don't want to print them twice
type SilentError struct {
	Underlying error
}

func (e SilentError) Error() string {
	return e.Underlying.Error()
}

func silentError(err error) SilentError {
	return SilentError{Underlying: err}
}

func skipTLSValidation(baseImageURL *url.URL, trustedRegistries []string) bool {
	for _, trustedRegistry := range trustedRegistries {
		if baseImageURL.Host == trustedRegistry {
			return true
		}
	}

	return false
}

func validateArgs(ctx *cli.Context, num int) error {
	if len(ctx.Args()) != num {
		return fmt.Errorf("Incorrect number of args. Expect %d, got %d", num, len(ctx.Args()))
	}

	return nil
}
