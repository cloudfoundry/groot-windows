module code.cloudfoundry.org/groot-windows

go 1.21

toolchain go1.21.8

replace github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.7

require (
	code.cloudfoundry.org/filelock v0.0.0-20240711155740-a50c183dc1e0
	code.cloudfoundry.org/groot v0.0.0-20240716202325-6c1c14d5995e
	code.cloudfoundry.org/hydrator v0.0.0-20240717160259-cf86605dd648
	code.cloudfoundry.org/lager/v3 v3.0.3
	github.com/Microsoft/go-winio v0.6.2
	github.com/Microsoft/hcsshim v0.12.5
	github.com/onsi/ginkgo/v2 v2.19.0
	github.com/onsi/gomega v1.33.1
	github.com/opencontainers/image-spec v1.1.0
	github.com/opencontainers/runtime-spec v1.2.0
	github.com/urfave/cli v1.22.15
	golang.org/x/sys v0.22.0
)

require (
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containers/image/v5 v5.31.1 // indirect
	github.com/containers/libtrust v0.0.0-20230121012942-c1716e8a8d01 // indirect
	github.com/containers/ocicrypt v1.2.0 // indirect
	github.com/containers/storage v1.54.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v27.0.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20240711041743-f6c9dda6c6da // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
