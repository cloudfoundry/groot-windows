module code.cloudfoundry.org/groot-windows

go 1.15

require (
	code.cloudfoundry.org/archiver v0.0.0-20210513174825-6979f8d756e2 // indirect
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/groot v0.0.0-20210505095527-8906ba001ae9
	code.cloudfoundry.org/hydrator v0.0.0-20180411234439-6b2757c7f6f0
	code.cloudfoundry.org/lager v1.1.0
	github.com/Microsoft/go-winio v0.4.7
	github.com/Microsoft/hcsshim v0.8.17
	github.com/docker/docker-credential-helpers v0.6.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/urfave/cli v1.22.2
	golang.org/x/net v0.0.0-20210521195947-fe42d452be8f // indirect
	golang.org/x/sys v0.0.0-20210521203332-0cec03c779c1
)

replace github.com/Microsoft/hcsshim v0.8.17 => github.com/greenhouse-org/hcsshim v0.6.8-0.20190130155644-d3cfe7c848cd
